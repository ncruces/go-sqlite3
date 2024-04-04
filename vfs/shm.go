//go:build (linux || darwin) && (amd64 || arm64) && !sqlite3_flock && !sqlite3_nosys

package vfs

import (
	"context"
	"io"
	"os"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

// SupportsSharedMemory is true on platforms that support shared memory.
// To enable shared memory support on those platforms,
// you need to set the appropriate [wazero.RuntimeConfig];
// otherwise, [EXCLUSIVE locking mode] is activated automatically
// to use [WAL without shared-memory].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = true

func vfsVersion(mod api.Module) uint32 {
	pagesize := unix.Getpagesize()

	// 32KB pages must be a multiple of the system's page size.
	if (32*1024)%pagesize != 0 {
		return 0
	}

	// The module's memory must be page aligned.
	b, ok := mod.Memory().Read(0, 1)
	if ok && uintptr(unsafe.Pointer(&b[0]))%uintptr(pagesize) != 0 {
		return 0
	}

	return 1 // TODO: feeling lucky?
}

type vfsShm struct {
	*os.File
	regions []*util.MappedRegion
}

const (
	_SHM_NLOCK = 8
	_SHM_BASE  = 120
	_SHM_DMS   = _SHM_BASE + _SHM_NLOCK
)

func (f *vfsFile) shmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (_ uint32, err error) {
	// Ensure size is a multiple of the OS page size.
	if int(size)%unix.Getpagesize() != 0 {
		return 0, _IOERR_SHMMAP
	}

	// TODO: handle the read-only case.
	if f.shm.File == nil {
		f.shm.File, err = os.OpenFile(f.Name()+"-shm", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW, 0666)
		if err != nil {
			return 0, _IOERR_SHMOPEN
		}
	}

	// Dead man's switch.
	if lock, rc := osGetLock(f.shm.File, _SHM_DMS, 1); rc != _OK {
		return 0, _IOERR_LOCK
	} else if lock == unix.F_WRLCK {
		return 0, _BUSY
	} else if lock == unix.F_UNLCK {
		if rc := osWriteLock(f.shm.File, _SHM_DMS, 1, 0); rc != _OK {
			return 0, rc
		}
		if err := f.shm.Truncate(0); err != nil {
			return 0, _IOERR_SHMOPEN
		}
	}
	if rc := osReadLock(f.shm.File, _SHM_DMS, 1, 0); rc != _OK {
		return 0, rc
	}

	// Check if file is big enough.
	s, err := f.shm.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, _IOERR_SHMSIZE
	}
	if n := (int64(id) + 1) * int64(size); n > s {
		if !extend {
			return 0, nil
		}
		err := osAllocate(f.shm.File, n)
		if err != nil {
			return 0, _IOERR_SHMOPEN
		}
	}

	r, err := util.MapRegion(ctx, mod, f.shm.File, int64(id)*int64(size), size)
	if err != nil {
		return 0, err
	}
	f.shm.regions = append(f.shm.regions, r)
	return r.Ptr, nil
}

func (f *vfsFile) shmLock(offset, n uint32, flags _ShmFlag) error {
	// Argument check.
	if n == 0 || offset+n > _SHM_NLOCK {
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
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(f.shm.File, _SHM_BASE+int64(offset), int64(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(f.shm.File, _SHM_BASE+int64(offset), int64(n), 0)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(f.shm.File, _SHM_BASE+int64(offset), int64(n), 0)
	default:
		panic(util.AssertErr())
	}
}

func (f *vfsFile) shmUnmap(delete bool) {
	// Unmap regions.
	for _, r := range f.shm.regions {
		r.Unmap()
	}
	clear(f.shm.regions)
	f.shm.regions = f.shm.regions[:0]

	// Close the file.
	if delete && f.shm.File != nil {
		os.Remove(f.shm.Name())
	}
	f.shm.Close()
	f.shm.File = nil
}
