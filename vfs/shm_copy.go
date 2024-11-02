//go:build sqlite3_dotlk

package vfs

import (
	"context"
	"sync"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

const _WALINDEX_PGSZ = 32768

type vfsShmBuffer struct {
	shared []byte // +checklocks:Mutex
	refs   int    // +checklocks:vfsShmBuffersMtx

	lock [_SHM_NLOCK]int16 // +checklocks:Mutex
	sync.Mutex
}

var (
	// +checklocks:vfsShmBuffersMtx
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
	if size != _WALINDEX_PGSZ {
		return 0, _IOERR_SHMMAP
	}
	if s.mod == nil {
		s.mod = mod
		s.free = mod.ExportedFunction("sqlite3_free")
		s.alloc = mod.ExportedFunction("sqlite3_malloc64")
	}

	s.shmOpen()
	s.Lock()
	defer s.Unlock()
	defer s.shmAcquire()

	n := (int(id) + 1) * int(size)

	if n > len(s.shared) {
		if !extend {
			return 0, _OK
		}
		s.shared = append(s.shared, make([]byte, n-len(s.shared))...)
	}

	if n > len(s.shadow) {
		s.shadow = append(s.shadow, make([]byte, n-len(s.shadow))...)
	}

	for int(id) >= len(s.ptrs) {
		s.stack[0] = uint64(size)
		if err := s.alloc.CallWithStack(ctx, s.stack[:]); err != nil {
			panic(err)
		}
		if s.stack[0] == 0 {
			panic(util.OOMErr)
		}
		clear(util.View(s.mod, uint32(s.stack[0]), _WALINDEX_PGSZ))
		s.ptrs = append(s.ptrs, uint32(s.stack[0]))
	}

	return s.ptrs[id], _OK
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) _ErrorCode {
	s.Lock()
	defer s.Unlock()

	switch {
	case flags&_SHM_LOCK != 0:
		s.shmAcquire()
	case flags&_SHM_EXCLUSIVE != 0:
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

// This looks like a safe, if inefficient, way of keeping memory in sync.
//
// The WAL-index file starts with a header.
// This header starts with two 48 byte, checksummed, copies of the same information,
// which are accessed independently between memory barriers.
// The checkpoint information that follows uses 4 byte aligned words.
//
// Finally, we have the WAL-index hash tables,
// which are only modified holding the exclusive WAL_WRITE_LOCK.
// Also, aHash isn't modified unless aPgno changes.
//
// Since all the data is either redundant+checksummed,
// 4 byte aligned, or modified under an exclusive lock,
// the copies below should correctly keep memory in sync.
//
// https://sqlite.org/walformat.html#the_wal_index_file_format

// +checklocks:s.Mutex
func (s *vfsShm) shmAcquire() {
	// Copies modified words from shared to private memory.
	for id, p := range s.ptrs {
		i0 := id * _WALINDEX_PGSZ
		i1 := i0 + _WALINDEX_PGSZ
		shared := shmPage(s.shared[i0:i1])
		shadow := shmPage(s.shadow[i0:i1])
		privat := shmPage(util.View(s.mod, p, _WALINDEX_PGSZ))
		if shmPageEq(shadow, shared) {
			continue
		}
		for i, shared := range shared {
			if shadow[i] != shared {
				shadow[i] = shared
				privat[i] = shared
			}
		}
	}
}

// +checklocks:s.Mutex
func (s *vfsShm) shmRelease() {
	// Copies modified words from private to shared memory.
	for id, p := range s.ptrs {
		i0 := id * _WALINDEX_PGSZ
		i1 := i0 + _WALINDEX_PGSZ
		shared := shmPage(s.shared[i0:i1])
		shadow := shmPage(s.shadow[i0:i1])
		privat := shmPage(util.View(s.mod, p, _WALINDEX_PGSZ))
		if shmPageEq(shadow, privat) {
			continue
		}
		for i, privat := range privat {
			if shadow[i] != privat {
				shadow[i] = privat
				shared[i] = privat
			}
		}
	}
}

func shmPage(s []byte) *[_WALINDEX_PGSZ / 4]uint32 {
	p := (*uint32)(unsafe.Pointer(unsafe.SliceData(s)))
	return (*[_WALINDEX_PGSZ / 4]uint32)(unsafe.Slice(p, _WALINDEX_PGSZ/4))
}

func shmPageEq(p1, p2 *[_WALINDEX_PGSZ / 4]uint32) bool {
	return *(*[_WALINDEX_PGSZ / 8]uint32)(p1[:]) == *(*[_WALINDEX_PGSZ / 8]uint32)(p2[:])
}
