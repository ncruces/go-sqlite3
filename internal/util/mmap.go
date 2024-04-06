//go:build (linux || darwin) && (amd64 || arm64) && !sqlite3_flock && !sqlite3_nosys

package util

import (
	"context"
	"os"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

type mmapState struct {
	regions []*MappedRegion
	enabled bool
}

func (s *mmapState) closeNotify() {
	for _, r := range s.regions {
		r.Close()
	}
	s.regions = nil
}

func CanMap(ctx context.Context) bool {
	s := ctx.Value(moduleKey{}).(*moduleState)
	return s.mmapState.enabled
}

func (s *mmapState) new(ctx context.Context, mod api.Module, size uint32) *MappedRegion {
	// Find unused region.
	for _, r := range s.regions {
		if !r.used && r.size == size {
			return r
		}
	}

	// Allocate page aligned memmory.
	alloc := mod.ExportedFunction("aligned_alloc")
	stack := [2]uint64{
		uint64(unix.Getpagesize()),
		uint64(size),
	}
	if err := alloc.CallWithStack(ctx, stack[:]); err != nil {
		panic(err)
	}
	if stack[0] == 0 {
		panic(OOMErr)
	}

	// Save the newly allocated region.
	ptr := uint32(stack[0])
	buf := View(mod, ptr, uint64(size))
	addr := uintptr(unsafe.Pointer(&buf[0]))
	s.regions = append(s.regions, &MappedRegion{
		Ptr:  ptr,
		addr: addr,
		size: size,
	})
	return s.regions[len(s.regions)-1]
}

type MappedRegion struct {
	addr uintptr
	Ptr  uint32
	size uint32
	used bool
}

func MapRegion(ctx context.Context, mod api.Module, f *os.File, offset int64, size uint32) (*MappedRegion, error) {
	s := ctx.Value(moduleKey{}).(*moduleState)
	r := s.new(ctx, mod, size)
	err := r.mmap(f, offset)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *MappedRegion) Close() error {
	return munmap(r.addr, uintptr(r.size))
}

func (r *MappedRegion) Unmap() error {
	// We can't munmap the region, otherwise it could be remaped.
	// Instead, convert it to a protected, private, anonymous mapping.
	// If successful, it can be reused for a subsequent mmap.
	_, err := mmap(r.addr, uintptr(r.size),
		unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_ANON|unix.MAP_FIXED,
		-1, 0)
	r.used = err != nil
	return err
}

func (r *MappedRegion) mmap(f *os.File, offset int64) error {
	_, err := mmap(r.addr, uintptr(r.size),
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_FIXED,
		int(f.Fd()), offset)
	r.used = err == nil
	return err
}

type mappableMemoryAllocator struct{}

func (mappableMemoryAllocator) Make(min, cap, max uint64) []byte {
	addr, err := mmap(0, uintptr(max),
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON,
		-1, 0)
	if err != nil {
		panic(OOMErr)
	}
	return unsafe.Slice(addr, max)[:min]
}

func (mappableMemoryAllocator) Grow(old []byte, more uint64) []byte {
	panic(OOMErr)
}

func (mappableMemoryAllocator) Free(buf []byte) {
	munmap(uintptr(unsafe.Pointer(&buf[0])), uintptr(cap(buf)))
}

//go:linkname mmap syscall.mmap
func mmap(addr, length uintptr, prot, flag, fd int, pos int64) (*byte, error)

//go:linkname munmap syscall.munmap
func munmap(addr, length uintptr) error