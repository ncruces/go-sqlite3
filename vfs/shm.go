package vfs

import (
	"context"
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

type vfsShm interface {
	ShmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (uint32, error)
	ShmLock() error
	ShmUnmap()
}

func (f *vfsFile) ShmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (uint32, error) {
	m, err := os.OpenFile(f.Name()+"-shm", os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0666)
	if err != nil {
		return 0, _IOERR_SHMOPEN
	}
	s, err := m.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, _IOERR_SHMSIZE
	}
	if n := (int64(id) + 1) * int64(size); n > s {
		if !extend {
			return 0, nil
		}
		err := osAllocate(m, n)
		if err != nil {
			return 0, _IOERR_SHMOPEN
		}
	}

	alloc := mod.ExportedFunction("aligned_alloc")
	stack := [2]uint64{
		uint64(syscall.Getpagesize()),
		uint64(size),
	}
	if err := alloc.CallWithStack(ctx, stack[:]); err != nil {
		panic(err)
	}
	b := util.View(mod, uint32(stack[0]), uint64(size))
	_, err = mmap(uintptr(unsafe.Pointer(&b[0])), uintptr(size),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
		int(m.Fd()), int64(id)*int64(size))
	if err != nil {
		return 0, _IOERR_SHMMAP
	}
	return uint32(stack[0]), nil
}

func (f *vfsFile) ShmLock() error {
	return _IOERR_SHMLOCK
}

func (f *vfsFile) ShmUnmap() {}

//go:uintptrescapes
//go:linkname mmap syscall.mmap
func mmap(addr uintptr, length uintptr, prot int, flag int, fd int, pos int64) (ret uintptr, err error)
