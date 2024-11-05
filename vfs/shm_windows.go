//go:build (386 || arm || amd64 || arm64 || riscv64 || ppc64le) && !(sqlite3_dotlk || sqlite3_nosys)

package vfs

import (
	"context"
	"io"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/windows"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/osutil"
)

type vfsShm struct {
	*os.File
	mod      api.Module
	alloc    api.Function
	free     api.Function
	path     string
	regions  []*util.MappedRegion
	shared   [][]byte
	shadow   []byte
	ptrs     []uint32
	stack    [1]uint64
	blocking bool
	sync.Mutex
}

var _ blockingSharedMemory = &vfsShm{}

func (s *vfsShm) Close() error {
	// Unmap regions.
	for _, r := range s.regions {
		r.Unmap()
	}
	s.regions = nil

	// Close the file.
	s.File = nil
	return s.File.Close()
}

func (s *vfsShm) shmOpen() _ErrorCode {
	if s.File == nil {
		f, err := osutil.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return _CANTOPEN
		}
		s.File = f
	}

	// Dead man's switch.
	if rc := osWriteLock(s.File, _SHM_DMS, 1, 0); rc == _OK {
		err := s.Truncate(0)
		osUnlock(s.File, _SHM_DMS, 1)
		if err != nil {
			return _IOERR_SHMOPEN
		}
	}
	return osReadLock(s.File, _SHM_DMS, 1, time.Millisecond)
}

func (s *vfsShm) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (uint32, _ErrorCode) {
	// Ensure size is a multiple of the OS page size.
	if size != _WALINDEX_PGSZ || (windows.Getpagesize()-1)&_WALINDEX_PGSZ != 0 {
		return 0, _IOERR_SHMMAP
	}
	if s.mod == nil {
		s.mod = mod
		s.free = mod.ExportedFunction("sqlite3_free")
		s.alloc = mod.ExportedFunction("sqlite3_malloc64")
	}
	if rc := s.shmOpen(); rc != _OK {
		return 0, rc
	}

	defer s.shmAcquire()

	// Check if file is big enough.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, _IOERR_SHMSIZE
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, _OK
		}
		if osAllocate(s.File, n) != nil {
			return 0, _IOERR_SHMSIZE
		}
	}

	// Map the file into memory.
	r, err := util.MapRegion(ctx, mod, s.File, int64(id)*int64(size), size)
	if err != nil {
		return 0, _IOERR_SHMMAP
	}
	s.regions = append(s.regions, r)

	if int(id) >= len(s.shared) {
		s.shared = append(s.shared, make([][]byte, int(id)-len(s.shared))...)
	}
	s.shared[id] = r.Data

	// Allocate shadow memory.
	if n := (int(id) + 1) * int(size); n > len(s.shadow) {
		s.shadow = append(s.shadow, make([]byte, n-len(s.shadow))...)
	}

	// Allocate local memory.
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
	// Argument check.
	if n <= 0 || offset < 0 || offset+n > _SHM_NLOCK {
		panic(util.AssertErr())
	}
	switch flags {
	case
		_SHM_LOCK | _SHM_SHARED,
		_SHM_LOCK | _SHM_EXCLUSIVE,
		_SHM_UNLOCK | _SHM_SHARED,
		_SHM_UNLOCK | _SHM_EXCLUSIVE:
		//
	default:
		panic(util.AssertErr())
	}
	if n != 1 && flags&_SHM_EXCLUSIVE == 0 {
		panic(util.AssertErr())
	}

	switch {
	case flags&_SHM_LOCK != 0:
		defer s.shmAcquire()
	case flags&_SHM_EXCLUSIVE != 0:
		s.shmRelease()
	}

	var timeout time.Duration
	if s.blocking {
		timeout = time.Millisecond
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(s.File, _SHM_BASE+uint32(offset), uint32(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(s.File, _SHM_BASE+uint32(offset), uint32(n), timeout)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(s.File, _SHM_BASE+uint32(offset), uint32(n), timeout)
	default:
		panic(util.AssertErr())
	}
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.File == nil {
		return
	}

	s.shmRelease()

	// Free local memory.
	for _, p := range s.ptrs {
		s.stack[0] = uint64(p)
		if err := s.free.CallWithStack(context.Background(), s.stack[:]); err != nil {
			panic(err)
		}
	}
	s.ptrs = nil
	s.shadow = nil
	s.shared = nil

	// Close the file.
	if delete {
		os.Remove(s.path)
	}
	s.Close()
	s.File = nil
}

func (s *vfsShm) shmBarrier() {
	s.Lock()
	s.shmAcquire()
	s.shmRelease()
	s.Unlock()
}

const _WALINDEX_PGSZ = 32768

func (s *vfsShm) shmAcquire() {
	// Copies modified words from shared to private memory.
	for id, p := range s.ptrs {
		i0 := id * _WALINDEX_PGSZ
		i1 := i0 + _WALINDEX_PGSZ
		shared := shmPage(s.shared[id])
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

func (s *vfsShm) shmRelease() {
	// Copies modified words from private to shared memory.
	for id, p := range s.ptrs {
		i0 := id * _WALINDEX_PGSZ
		i1 := i0 + _WALINDEX_PGSZ
		shared := shmPage(s.shared[id])
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

func (s *vfsShm) shmEnableBlocking(block bool) {
	s.blocking = block
}
