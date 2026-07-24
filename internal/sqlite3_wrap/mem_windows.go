package sqlite3_wrap

import (
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Memory struct {
	Buf []byte
	Max int64
	com int
	ptr uintptr
	pcs []uintptr
}

func (m *Memory) Slice() *[]byte {
	return &m.Buf
}

func (m *Memory) Grow(delta, max int64) int64 {
	if m.Buf == nil {
		m.allocate(uint64(m.Max) << 16)
	}

	len := int64(len(m.Buf))
	old := len >> 16
	if delta == 0 {
		return old
	}
	new := old + delta
	max = min(max, m.Max, int64(math.MaxInt)>>16)
	if new > max || new < old {
		return -1
	}
	m.reallocate(uint64(new) << 16)
	return old
}

func (m *Memory) allocate(max uint64) {
	if !placeholdersSupported() {
		return
	}

	// Round up to the allocation granularity.
	rnd := uint64(allocationGranularity - 1)
	res := (max + rnd) &^ rnd

	if res > math.MaxInt {
		// This ensures uintptr(res) overflows to a large value,
		// and VirtualAlloc2 returns an error.
		res = math.MaxUint64
	}

	// Reserve res bytes of address space, to ensure we won't need to move it.
	// Use virtual memory placeholders so we can later map files.
	// https://devblogs.microsoft.com/oldnewthing/?p=109346
	r, err := virtualAlloc2(0, uintptr(res),
		windows.MEM_RESERVE|_MEM_RESERVE_PLACEHOLDER, windows.PAGE_NOACCESS)
	if err != nil {
		panic(err)
	}
	m.pcs = append(m.pcs, 0)
	m.ptr = r

	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&m.ptr))
	m.Buf = unsafe.Slice((*byte)(ptr), res)[:0]
}

func (m *Memory) reallocate(size uint64) {
	if m.ptr == 0 {
		m.Buf = append(m.Buf, make([]byte, size-uint64(len(m.Buf)))...)
		return
	}

	com := uint64(m.com)
	res := uint64(cap(m.Buf))
	if com < size && size <= res {
		// Round up to the allocation granularity.
		rnd := uint64(allocationGranularity - 1)
		new := (size + rnd) &^ rnd

		// Split the trailing placeholder.
		if new < res {
			err := windows.VirtualFree(m.ptr+uintptr(com), uintptr(new-com),
				windows.MEM_RELEASE|_MEM_PRESERVE_PLACEHOLDER)
			if err != nil {
				panic(err)
			}
			m.pcs = append(m.pcs, uintptr(new))
		}
		// Replace the placeholder with committed memory.
		_, err := virtualAlloc2(m.ptr+uintptr(com), uintptr(new-com),
			windows.MEM_COMMIT|windows.MEM_RESERVE|_MEM_REPLACE_PLACEHOLDER, windows.PAGE_READWRITE)
		if err != nil {
			panic(err)
		}
		m.com = int(new)
	}
	m.Buf = m.Buf[:size]
}

func (m *Memory) Close() error {
	if m.ptr == 0 {
		m.Buf = nil
		return nil
	}

	for i, off := range m.pcs {
		err := windows.VirtualFree(m.ptr+off, 0, windows.MEM_RELEASE)
		if err != nil {
			var next uintptr
			if i+1 < len(m.pcs) {
				next = m.pcs[i+1]
			} else {
				next = uintptr(cap(m.Buf))
			}
			if windows.VirtualFree(m.ptr+off, next-off, windows.MEM_RELEASE) != nil {
				panic(err)
			}
		}
	}

	m.Buf = nil
	m.com = 0
	m.ptr = 0
	m.pcs = nil
	return nil
}

// CanMapFiles reports whether file views can be mapped into this memory.
func (m *Memory) CanMapFiles() bool { return m.ptr != 0 }
