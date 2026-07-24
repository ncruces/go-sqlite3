package sqlite3_wrap

import (
	"math"
	"os"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"golang.org/x/sys/windows"
)

type mmapState struct {
	regions []*MappedRegion
}

func (s *mmapState) unmapAll() {
	for _, r := range s.regions {
		if r.used {
			r.Unmap()
		}
	}
}

func (w *Wrapper) MapRegion(f *os.File, offset int64, size int32, readOnly bool) (*MappedRegion, error) {
	pageSize := int64(allocationGranularity)
	align := offset & (pageSize - 1)
	offset -= align

	size += int32(align + pageSize - 1)
	size &^= int32(pageSize - 1)

	r := w.newRegion(size)
	err := r.mmap(f, offset, readOnly)
	if err != nil {
		return nil, err
	}
	r.Ptr = r.base + Ptr_t(align)
	return r, nil
}

func (w *Wrapper) newRegion(size int32) *MappedRegion {
	// Find unused region.
	for _, r := range w.regions {
		if !r.used && r.size == size {
			return r
		}
	}

	// Allocate page aligned memmory.
	rnd := int64(allocationGranularity - 1)
	new := (int64(size) + rnd) / allocationGranularity
	old := w.Memory.Grow(new, math.MaxInt64)
	if old < 0 {
		panic(errutil.OOMErr)
	}
	ptr := Ptr_t(old) * allocationGranularity

	// Save the newly allocated region.
	ret := &MappedRegion{
		base: ptr,
		size: size,
		addr: uintptr(unsafe.Pointer(&w.Buf[ptr])),
	}
	w.regions = append(w.regions, ret)
	return ret
}

type MappedRegion struct {
	addr        uintptr
	base        Ptr_t
	Ptr         Ptr_t
	size        int32
	used        bool
	placeholder bool
}

func (r *MappedRegion) Unmap() error {
	if !r.used {
		return nil
	}
	// Convert the region back to a placeholder.
	// If successful, it can be reused for a subsequent mmap.
	err := unmapViewOfFile2(r.addr, _MEM_PRESERVE_PLACEHOLDER)
	r.used = err != nil
	return err
}

func (r *MappedRegion) mmap(f *os.File, offset int64, readOnly bool) error {
	prot := uint32(windows.PAGE_READWRITE)
	if readOnly {
		prot = windows.PAGE_READONLY
	}
	err := mmap(f, offset, r.addr, r.size, prot, &r.placeholder)
	r.used = err == nil
	return err
}

func mmap(f *os.File, offset int64, addr uintptr, size int32, prot uint32, placeholder *bool) error {
	if !*placeholder {
		err := windows.VirtualFree(addr, uintptr(size), windows.MEM_RELEASE|_MEM_PRESERVE_PLACEHOLDER)
		if err != nil {
			return err
		}
		*placeholder = true
	}

	maxSize := offset + int64(size)

	h, err := windows.CreateFileMapping(
		windows.Handle(f.Fd()), nil, prot,
		uint32(maxSize>>32), uint32(maxSize), nil)
	if h == 0 {
		return err
	}
	defer windows.CloseHandle(h)

	_, err = mapViewOfFile3(h, addr, uint64(offset), uintptr(size),
		_MEM_REPLACE_PLACEHOLDER, prot)
	return err
}
