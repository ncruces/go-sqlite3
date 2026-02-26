package alloc

import (
	"math"
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The slice covers the entire mmapped memory:
//   - len(buf) is the already committed memory,
//   - cap(buf) is the reserved address space.
type Memory struct {
	buf  []byte
	addr uintptr
	Max  int32
}

func (m *Memory) Grow(mem *[]byte, delta, _ int32) int32 {
	if m.buf == nil {
		m.allocate(uint64(m.Max) << 16)
	}

	buf := *mem
	len := len(buf)
	old := int32(len >> 16)
	if delta == 0 {
		return old
	}
	new := old + delta
	if new > m.Max {
		return -1
	}
	*mem = m.reallocate(uint64(new) << 16)
	return old
}

func (m *Memory) allocate(max uint64) {
	// Round up to the page size.
	rnd := uint64(windows.Getpagesize() - 1)
	res := (max + rnd) &^ rnd

	if res > math.MaxInt {
		// This ensures uintptr(res) overflows to a large value,
		// and windows.VirtualAlloc returns an error.
		res = math.MaxUint64
	}

	// Reserve res bytes of address space, to ensure we won't need to move it.
	r, err := windows.VirtualAlloc(0, uintptr(res), uint32(windows.MEM_RESERVE), windows.PAGE_READWRITE)
	if err != nil {
		panic(err)
	}

	// SliceHeader, although deprecated, avoids a go vet warning.
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&m.buf))
	sh.Data = r
	sh.Len = 0
	sh.Cap = int(res)
	m.addr = r
}

func (m *Memory) reallocate(size uint64) []byte {
	com := uint64(len(m.buf))
	res := uint64(cap(m.buf))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(windows.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		_, err := windows.VirtualAlloc(m.addr, uintptr(new), windows.MEM_COMMIT, windows.PAGE_READWRITE)
		if err != nil {
			return nil
		}

		m.buf = m.buf[:new] // Update committed memory.
	}
	// Limit returned capacity because bytes beyond
	// len(m.buf) have not yet been committed.
	return m.buf[:size:len(m.buf)]
}

func (m *Memory) Free() {
	err := windows.VirtualFree(m.addr, 0, windows.MEM_RELEASE)
	if err != nil {
		panic(err)
	}
	m.addr = 0
}
