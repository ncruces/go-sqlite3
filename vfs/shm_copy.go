//go:build sqlite3_dotlk

package vfs

import (
	"context"
	"sync"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

const _SHM_NLOCK = 8

type vfsShmBuffer struct {
	data []byte
	refs int

	lock [_SHM_NLOCK]int16
	sync.Mutex
}

var (
	vfsShmBuffers    = map[string]*vfsShmBuffer{}
	vfsShmBuffersMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmBuffer
	mod      api.Module
	alloc    api.Function
	free     api.Function
	path     string
	shadow   []byte
	ptrs     []uint32
	stack    [1]uint64
	lock     [_SHM_NLOCK]bool
	size     int32
	readOnly bool
}

func (s *vfsShm) Close() error {
	if s.vfsShmBuffer == nil {
		return nil
	}

	vfsShmBuffersMtx.Lock()
	defer vfsShmBuffersMtx.Unlock()

	// Unlock everything.
	s.shmLock(0, _SHM_NLOCK, _SHM_UNLOCK)

	// Decrease reference count.
	if s.vfsShmBuffer.refs > 0 {
		s.vfsShmBuffer.refs--
		s.vfsShmBuffer = nil
		return nil
	}

	delete(vfsShmBuffers, s.path)
	return nil
}

func (s *vfsShm) shmOpen() {
	if s.vfsShmBuffer != nil {
		return
	}

	vfsShmBuffersMtx.Lock()
	defer vfsShmBuffersMtx.Unlock()

	// Find a shared buffer, increase the reference count.
	if g, ok := vfsShmBuffers[s.path]; ok {
		s.vfsShmBuffer = g
		g.refs++
		return
	}

	// Add the new shared buffer.
	s.vfsShmBuffer = &vfsShmBuffer{}
	vfsShmBuffers[s.path] = s.vfsShmBuffer
}

func (s *vfsShm) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (uint32, _ErrorCode) {
	switch {
	case s.size == 0:
		s.size = size
		s.mod = mod
		s.free = mod.ExportedFunction("sqlite3_free")
		s.alloc = mod.ExportedFunction("sqlite3_malloc64")
	case s.size != size:
		return 0, _IOERR_SHMMAP
	}

	s.shmOpen()
	s.Lock()
	defer s.Unlock()
	defer s.shmAcquire()

	n := (int(id) + 1) * int(size)

	if n > len(s.data) {
		if !extend {
			return 0, _OK
		}
		s.data = append(s.data, make([]byte, n-len(s.data))...)
	}

	if n > len(s.shadow) {
		s.shadow = append(s.shadow, make([]byte, n-len(s.shadow))...)
	}

	if int(id) == len(s.ptrs) {
		s.stack[0] = uint64(size)
		if err := s.alloc.CallWithStack(ctx, s.stack[:]); err != nil {
			panic(err)
		}
		if s.stack[0] == 0 {
			panic(util.OOMErr)
		}
		clear(util.View(s.mod, uint32(s.stack[0]), uint64(s.size)))
		s.ptrs = append(s.ptrs, uint32(s.stack[0]))
	}

	return s.ptrs[id], _OK
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) _ErrorCode {
	s.Lock()
	defer s.Unlock()

	if flags&_SHM_UNLOCK == 0 {
		s.shmAcquire()
	} else {
		s.shmRelease()
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				if s.vfsShmBuffer.lock[i] == 0 {
					panic(util.AssertErr())
				}
				if s.vfsShmBuffer.lock[i] <= 0 {
					s.vfsShmBuffer.lock[i] = 0
				} else {
					s.vfsShmBuffer.lock[i]--
				}
				s.lock[i] = false
			}
		}
	case flags&_SHM_SHARED != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				panic(util.AssertErr())
			}
			if s.vfsShmBuffer.lock[i]+1 <= 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmBuffer.lock[i]++
			s.lock[i] = true
		}
	case flags&_SHM_EXCLUSIVE != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				panic(util.AssertErr())
			}
			if s.vfsShmBuffer.lock[i] != 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmBuffer.lock[i] = -1
			s.lock[i] = true
		}
	default:
		panic(util.AssertErr())
	}

	return _OK
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.vfsShmBuffer == nil {
		return
	}
	defer s.Close()

	s.Lock()
	s.shmRelease()
	defer s.Unlock()

	for _, p := range s.ptrs {
		s.stack[0] = uint64(p)
		if err := s.free.CallWithStack(context.Background(), s.stack[:]); err != nil {
			panic(err)
		}
	}
	s.ptrs = nil
	s.shadow = nil
}

func (s *vfsShm) shmBarrier() {
	s.Lock()
	s.shmAcquire()
	s.shmRelease()
	s.Unlock()
}

func (s *vfsShm) shmAcquire() {
	// Copies modified words from shared memory to local memory.
	for id, p := range s.ptrs {
		i0 := id * int(s.size)
		i1 := i0 + int(s.size)
		data := shmWords(s.data[i0:i1])
		shadow := shmWords(s.shadow[i0:i1])[:len(data)]
		local := shmWords(util.View(s.mod, p, uint64(s.size)))[:len(data)]
		for i, data := range data {
			if shadow[i] != data {
				shadow[i] = data
				local[i] = data
			}
		}
	}
}

func (s *vfsShm) shmRelease() {
	// Copies modified words from local memory to shared memory.
	for id, p := range s.ptrs {
		i0 := id * int(s.size)
		i1 := i0 + int(s.size)
		data := shmWords(s.data[i0:i1])
		shadow := shmWords(s.shadow[i0:i1])[:len(data)]
		local := shmWords(util.View(s.mod, p, uint64(s.size)))[:len(data)]
		for i, local := range local {
			if shadow[i] != local {
				shadow[i] = local
				data[i] = local
			}
		}
	}
}

func shmWords(s []uint8) []uint32 {
	p := unsafe.SliceData(s)
	return unsafe.Slice((*uint32)(unsafe.Pointer(p)), len(s)/4)
}
