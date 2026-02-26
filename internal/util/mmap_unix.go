//go:build unix

package util

import (
	"context"
	"os"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/sqlite3_wasm"
	"golang.org/x/sys/unix"
)

type mmapState struct {
	regions []*MappedRegion
}

func (s *mmapState) new(_ context.Context, mod *sqlite3_wasm.Module, size int32) *MappedRegion {
	// Find unused region.
	for _, r := range s.regions {
		if !r.used && r.size == size {
			return r
		}
	}

	// Allocate page aligned memmory.
	ptr := Ptr_t(mod.Xaligned_alloc(int32(unix.Getpagesize()), size))
	if ptr == 0 {
		panic(OOMErr)
	}

	// Save the newly allocated region.
	buf := View(mod, ptr, int64(size))
	ret := &MappedRegion{
		Ptr:  ptr,
		size: size,
		addr: unsafe.Pointer(&buf[0]),
	}
	s.regions = append(s.regions, ret)
	return ret
}

type MappedRegion struct {
	addr unsafe.Pointer
	Ptr  Ptr_t
	size int32
	used bool
}

func MapRegion(ctx context.Context, mod *sqlite3_wasm.Module, f *os.File, offset int64, size int32, readOnly bool) (*MappedRegion, error) {
	s := ctx.Value(moduleKey{}).(*moduleState)
	r := s.new(ctx, mod, size)
	err := r.mmap(f, offset, readOnly)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *MappedRegion) Unmap() error {
	// We can't munmap the region, otherwise it could be remaped.
	// Instead, convert it to a protected, private, anonymous mapping.
	// If successful, it can be reused for a subsequent mmap.
	_, err := unix.MmapPtr(-1, 0, r.addr, uintptr(r.size),
		unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_FIXED|unix.MAP_ANON)
	r.used = err != nil
	return err
}

func (r *MappedRegion) mmap(f *os.File, offset int64, readOnly bool) error {
	prot := unix.PROT_READ
	if !readOnly {
		prot |= unix.PROT_WRITE
	}
	_, err := unix.MmapPtr(int(f.Fd()), offset, r.addr, uintptr(r.size),
		prot, unix.MAP_SHARED|unix.MAP_FIXED)
	r.used = err == nil
	return err
}
