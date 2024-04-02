//go:build (linux || darwin) && (amd64 || arm64) && !sqlite3_nosys

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

type vfsShm struct {
	*os.File
	regions []shmRegion
}

type shmRegion struct {
	addr   uintptr
	length uint32
}

const (
	_SHM_NLOCK = 8
	_SHM_BASE  = 120
	_SHM_DMS   = _SHM_BASE + _SHM_NLOCK
)

func (f *vfsFile) ShmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (_ uint32, err error) {
	// Ensure size is a multiple of the OS page size.
	// TODO: don't implement shared memory if this isn't the case.
	if int(size)%unix.Getpagesize() != 0 {
		return 0, _IOERR_SHMMAP
	}

	// TODO: handle the read-only case.
	// TODO: should we close the file on error?
	if f.shm.File == nil {
		f.shm.File, err = os.OpenFile(f.Name()+"-shm", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW, 0666)
		if err != nil {
			return 0, _IOERR_SHMOPEN
		}
	}

	// Dead man's switch.
	// TODO: fix race condition.
	if rc := osReadLock(f.shm.File, _SHM_DMS, 1, 0); rc != _OK {
		return 0, rc
	}
	if rc := osWriteLock(f.shm.File, _SHM_DMS, 1, 0); rc == _OK {
		if err := f.shm.File.Truncate(0); err != nil {
			return 0, _IOERR_SHMOPEN
		}
		osReadLock(f.shm.File, _SHM_DMS, 1, 0)
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

	// Allocate some page aligned memmory.
	alloc := mod.ExportedFunction("aligned_alloc")
	stack := [2]uint64{
		uint64(unix.Getpagesize()),
		uint64(size),
	}
	if err := alloc.CallWithStack(ctx, stack[:]); err != nil {
		panic(err)
	}
	if stack[0] == 0 {
		panic(util.OOMErr)
	}

	// Map the file into the allocated pages.
	p := util.View(mod, uint32(stack[0]), uint64(size))
	a, err := mmap(uintptr(unsafe.Pointer(&p[0])), uintptr(size),
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_FIXED,
		int(f.shm.Fd()), int64(id)*int64(size))
	if err != nil {
		return 0, _IOERR_SHMMAP
	}

	f.shm.regions = append(f.shm.regions, shmRegion{a, size})
	return uint32(stack[0]), nil
}

func (f *vfsFile) ShmLock(offset, n uint32, flags _ShmFlag) error {
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

func (f *vfsFile) ShmUnmap(delete bool) {
	// TODO: recycle the malloc'd memory pages.

	// Unmap pages.
	for _, r := range f.shm.regions {
		// mmap(r.addr, uintptr(r.length), unix.PROT_NONE, unix.MAP_ANON|unix.MAP_FIXED, -1, 0)
		munmap(r.addr, uintptr(r.length))
	}
	f.shm.regions = f.shm.regions[:0]

	// Close the file.
	if f.shm.File != nil {
		if delete {
			os.Remove(f.shm.Name())
		}
		f.shm.Close()
		f.shm.File = nil
	}
}

//go:linkname mmap syscall.mmap
func mmap(addr, length uintptr, prot, flag, fd int, pos int64) (uintptr, error)

//go:linkname munmap syscall.munmap
func munmap(addr, length uintptr) error
