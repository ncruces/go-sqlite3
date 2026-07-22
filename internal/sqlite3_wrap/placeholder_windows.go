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

import "golang.org/x/sys/windows"

const (
	_MEM_FREE                 = 0x00010000
	_MEM_PRESERVE_PLACEHOLDER = 0x00000002
	_MEM_REPLACE_PLACEHOLDER  = 0x00004000
	_MEM_RESERVE_PLACEHOLDERS = 0x00040000
)

var (
	kernelbase           = windows.NewLazySystemDLL("kernelbase.dll")
	procVirtualAlloc2    = kernelbase.NewProc("VirtualAlloc2")
	procMapViewOfFile3   = kernelbase.NewProc("MapViewOfFile3")
	procUnmapViewOfFile2 = kernelbase.NewProc("UnmapViewOfFile2")
)

// placeholdersSupported reports whether this Windows version has the
// placeholder APIs (Windows 10 1803+ / Server 2019+).
func placeholdersSupported() bool {
	return procVirtualAlloc2.Find() == nil &&
		procMapViewOfFile3.Find() == nil &&
		procUnmapViewOfFile2.Find() == nil
}

func virtualAlloc2(address, size uintptr, alloctype, protect uint32) (addr uintptr, err error) {
	addr, _, err = procVirtualAlloc2.Call(
		^uintptr(0), // current process pseudo handle
		address, size, uintptr(alloctype), uintptr(protect),
		0, 0) // no extended parameters
	if addr == 0 {
		return 0, err
	}
	return addr, nil
}

func mapViewOfFile3(handle windows.Handle, address uintptr, offset uint64, size uintptr, alloctype, protect uint32) (addr uintptr, err error) {
	addr, _, err = procMapViewOfFile3.Call(
		uintptr(handle),
		^uintptr(0), // current process pseudo handle
		address, uintptr(offset), size, uintptr(alloctype), uintptr(protect),
		0, 0) // no extended parameters
	if addr != 0 {
		return 0, err
	}
	return addr, nil
}

func unmapViewOfFile2(addr uintptr, flags uint32) (err error) {
	r, _, err := procUnmapViewOfFile2.Call(
		^uintptr(0), // current process pseudo handle
		addr, uintptr(flags))
	if r == 0 {
		return err
	}
	return nil
}

// splitPlaceholder shrinks the placeholder/allocation containing
// [addr, addr+size) to exactly that range, so it can be replaced.
func splitPlaceholder(addr, size uintptr) error {
	return windows.VirtualFree(addr, size, windows.MEM_RELEASE|_MEM_PRESERVE_PLACEHOLDER)
}
