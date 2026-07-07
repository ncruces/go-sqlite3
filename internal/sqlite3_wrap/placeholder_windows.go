package sqlite3_wrap

// Address-space placeholders (Windows 10 1803+ / Server 2019+) let a file
// view be mapped INTO the wasm linear memory, the same way the unix build
// maps the WAL-index with MAP_FIXED. SQLite then works on genuinely shared
// memory: no private copies, no sync points, native memory semantics.
//
// The round-trip used here:
//
//	reserve:  VirtualAlloc2(MEM_RESERVE|MEM_RESERVE_PLACEHOLDERS)
//	commit:   split placeholder, VirtualAlloc2(MEM_REPLACE_PLACEHOLDER|COMMIT)
//	carve:    split a committed (replaced-placeholder) range back into a
//	          placeholder with VirtualFree(MEM_RELEASE|MEM_PRESERVE_PLACEHOLDER)
//	map:      MapViewOfFile3(MEM_REPLACE_PLACEHOLDER) into the carved hole
//	unmap:    UnmapViewOfFile2(MEM_PRESERVE_PLACEHOLDER), then re-commit

import (
	"golang.org/x/sys/windows"
)

const (
	_MEM_COMMIT               = 0x00001000
	_MEM_RESERVE              = 0x00002000
	_MEM_RELEASE              = 0x00008000
	_MEM_FREE                 = 0x00010000
	_MEM_RESERVE_PLACEHOLDERS = 0x00040000
	_MEM_REPLACE_PLACEHOLDER  = 0x00004000
	_MEM_PRESERVE_PLACEHOLDER = 0x00000002
	_PAGE_READWRITE           = 0x04
	_PAGE_NOACCESS            = 0x01
)

var (
	kernelbase           = windows.NewLazySystemDLL("kernelbase.dll")
	procVirtualAlloc2    = kernelbase.NewProc("VirtualAlloc2")
	procMapViewOfFile3   = kernelbase.NewProc("MapViewOfFile3")
	procUnmapViewOfFile2 = kernelbase.NewProc("UnmapViewOfFile2")
)

// PlaceholdersSupported reports whether this Windows version has the
// placeholder APIs (Windows 10 1803+ / Server 2019+).
func PlaceholdersSupported() bool {
	return procVirtualAlloc2.Find() == nil &&
		procMapViewOfFile3.Find() == nil &&
		procUnmapViewOfFile2.Find() == nil
}

func virtualAlloc2(addr uintptr, size uintptr, allocType, protect uint32) (uintptr, error) {
	r, _, err := procVirtualAlloc2.Call(
		0, // current process
		addr,
		size,
		uintptr(allocType),
		uintptr(protect),
		0, 0) // no extended parameters
	if r == 0 {
		return 0, err
	}
	return r, nil
}

func mapViewOfFile3(h windows.Handle, addr uintptr, offset uint64, size uintptr, allocType, protect uint32) (uintptr, error) {
	r, _, err := procMapViewOfFile3.Call(
		uintptr(h),
		0, // current process
		addr,
		uintptr(offset),
		size,
		uintptr(allocType),
		uintptr(protect),
		0, 0)
	if r == 0 {
		return 0, err
	}
	return r, nil
}

func unmapViewOfFile2(addr uintptr, unmapFlags uint32) error {
	r, _, err := procUnmapViewOfFile2.Call(
		^uintptr(0), // current process pseudo handle
		addr,
		uintptr(unmapFlags))
	if r == 0 {
		return err
	}
	return nil
}

// splitPlaceholder shrinks the placeholder/allocation containing
// [addr, addr+size) to exactly that range, so it can be replaced.
func splitPlaceholder(addr, size uintptr) error {
	return windows.VirtualFree(addr, size, _MEM_RELEASE|_MEM_PRESERVE_PLACEHOLDER)
}
