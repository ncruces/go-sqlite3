package sqlite3_wrap

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type MappedRegion struct {
	Data []byte
	addr uintptr
}

func MapRegion(f *os.File, offset int64, size int32) (*MappedRegion, error) {
	maxSize := offset + int64(size)
	h, err := windows.CreateFileMapping(
		windows.Handle(f.Fd()), nil, windows.PAGE_READWRITE,
		uint32(maxSize>>32), uint32(maxSize), nil)
	if h == 0 {
		return nil, err
	}
	defer windows.CloseHandle(h)

	const allocationGranularity = 64 * 1024
	align := offset % allocationGranularity
	offset -= align

	a, err := windows.MapViewOfFile(h, windows.FILE_MAP_WRITE,
		uint32(offset>>32), uint32(offset), uintptr(size)+uintptr(align))
	if a == 0 {
		return nil, err
	}

	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&a))
	return &MappedRegion{
		Data: unsafe.Slice((*byte)(unsafe.Add(ptr, align)), size),
		addr: a,
	}, nil
}

func (r *MappedRegion) Unmap() error {
	if r.Data == nil {
		return nil
	}
	err := windows.UnmapViewOfFile(r.addr)
	if err != nil {
		return err
	}
	r.Data = nil
	return nil
}
