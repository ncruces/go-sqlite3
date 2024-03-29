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

type vfsShm []vfsShmRegion

type vfsShmRegion struct {
	addr   uintptr
	length uint32
}

func (f *vfsFile) ShmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (uint32, error) {
	if unix.Getpagesize() > int(size) {
		return 0, _IOERR_SHMMAP
	}

	m, err := os.OpenFile(f.Name()+"-shm", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW, 0666)
	if err != nil {
		return 0, _IOERR_SHMOPEN
	}
	defer m.Close()

	// Check if file is big enough.
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

	// Allocate some page aligned memmory.
	alloc := mod.ExportedFunction("aligned_alloc")
	stack := [2]uint64{
		uint64(unix.Getpagesize()),
		uint64(size),
	}
	if err := alloc.CallWithStack(ctx, stack[:]); err != nil {
		panic(err)
	}

	// Map the file into the allocated pages.
	p := util.View(mod, uint32(stack[0]), uint64(size))
	a, err := mmap(uintptr(unsafe.Pointer(&p[0])), uintptr(size),
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED,
		int(m.Fd()), int64(id)*int64(size))
	if err != nil {
		return 0, _IOERR_SHMMAP
	}

	f.shm = append(f.shm, vfsShmRegion{a, size})
	return uint32(stack[0]), nil
}

func (f *vfsFile) ShmLock() error {
	return _IOERR_SHMLOCK
}

func (f *vfsFile) ShmUnmap() {
	// TODO: recycle the malloc'd memory pages.
	for _, r := range f.shm {
		munmap(r.addr, uintptr(r.length))
	}
	f.shm = f.shm[:0]
}

//go:linkname mmap syscall.mmap
func mmap(addr, length uintptr, prot, flag, fd int, pos int64) (uintptr, error)

//go:linkname munmap syscall.munmap
func munmap(addr, length uintptr) error
