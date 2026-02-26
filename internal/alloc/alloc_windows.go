package alloc

import (
	"math"
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The mem slice covers the entire mapped memory:
//   - len(mem) is the already committed memory,
//   - cap(mem) is the reserved address space.
type Memory struct {
	Max  int32
	Buf  []byte
	mem  []byte
	addr uintptr
}

func (m *Memory) Data() *[]byte {
	return &m.Buf
}

func (m *Memory) Grow(delta, _ int32) int32 {
	if m.Buf == nil {
		m.allocate(uint64(m.Max) << 16)
	}

	len := len(m.Buf)
	old := int32(len >> 16)
	if delta == 0 {
		return old
	}
	new := old + delta
	if new > m.Max {
		return -1
	}
	m.reallocate(uint64(new) << 16)
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
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&m.mem))
	sh.Data = r
	sh.Len = 0
	sh.Cap = int(res)
	m.addr = r
}

func (m *Memory) reallocate(size uint64) {
	com := uint64(len(m.mem))
	res := uint64(cap(m.mem))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(windows.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		_, err := windows.VirtualAlloc(m.addr, uintptr(new), windows.MEM_COMMIT, windows.PAGE_READWRITE)
		if err != nil {
			panic(err)
		}

		m.mem = m.mem[:new] // Update committed memory.
	}
	// Limit returned capacity because bytes beyond
	// len(m.mem) have not yet been committed.
	m.Buf = m.mem[:size:len(m.mem)]
}

func (m *Memory) Free() {
	err := windows.VirtualFree(m.addr, 0, windows.MEM_RELEASE)
	if err != nil {
		panic(err)
	}
	m.addr = 0
}
