package sqlite3_wrap

import (
	"math"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Memory struct {
	Buf    []byte
	Max    int64
	com    int
	ptr    uintptr
	placeh bool      // reserved with placeholders (Win10 1803+)
	views  []uintptr // offsets of live file views, for Close
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
	// Round up to the page size.
	rnd := uint64(windows.Getpagesize() - 1)
	res := (max + rnd) &^ rnd

	if res > math.MaxInt {
		// This ensures uintptr(res) overflows to a large value,
		// and the reservation fails.
		res = math.MaxUint64
	}

	// Reserve res bytes of address space, to ensure we won't need to move it.
	// Prefer a placeholder reservation (Windows 10 1803+): it can later have
	// file views mapped into it, which the WAL-index shared memory uses.
	m.placeh = placeholdersSupported()
	if m.placeh {
		r, err := virtualAlloc2(0, uintptr(res),
			windows.MEM_RESERVE|_MEM_RESERVE_PLACEHOLDERS, windows.PAGE_NOACCESS)
		if err != nil {
			panic(err)
		}
		m.ptr = r
	} else {
		r, err := windows.VirtualAlloc(0, uintptr(res), windows.MEM_RESERVE, windows.PAGE_READWRITE)
		if err != nil {
			panic(err)
		}
		m.ptr = r
	}

	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&m.ptr))
	m.Buf = unsafe.Slice((*byte)(ptr), res)[:0]
}

func (m *Memory) reallocate(size uint64) {
	com := uint64(m.com)
	res := uint64(cap(m.Buf))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(windows.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		if m.placeh {
			// Round to the 64K allocation granularity so every committed
			// chunk boundary is 64K-aligned. A wal-index view (64K-aligned,
			// 64K) can then never straddle two allocations, which a
			// placeholder split cannot cross.
			const gran = 64 * 1024
			new = (new + gran - 1) &^ (gran - 1)
			new = min(new, res)
			// Split the trailing placeholder to [com, new) and replace it
			// with committed memory. Placeholder-lineage allocations can be
			// carved back into placeholders later, which file-view mapping
			// relies on.
			if new < res {
				if err := splitPlaceholder(m.ptr+uintptr(com), uintptr(new-com)); err != nil {
					panic(err)
				}
			}
			if _, err := virtualAlloc2(m.ptr+uintptr(com), uintptr(new-com),
				windows.MEM_RESERVE|windows.MEM_COMMIT|_MEM_REPLACE_PLACEHOLDER, windows.PAGE_READWRITE); err != nil {
				panic(err)
			}
		} else {
			// Commit additional memory up to new bytes.
			if _, err := windows.VirtualAlloc(m.ptr, uintptr(new), windows.MEM_COMMIT, windows.PAGE_READWRITE); err != nil {
				panic(err)
			}
		}
		m.com = int(new)
	}
	m.Buf = m.Buf[:size]
}

// CanMapFiles reports whether file views can be mapped into this memory.
func (m *Memory) CanMapFiles() bool { return m.placeh }

// MapFileRegion maps size bytes of f at offset fileOff into the linear
// memory at offset addrOff, replacing previously committed private memory.
// addrOff must be allocation-granularity aligned and [addrOff, addrOff+size)
// must lie within a single committed chunk (e.g. one sqlite3_malloc block
// well inside the heap). The caller owns making the range unused by Go/wasm
// code for the lifetime of the view.
func (m *Memory) MapFileRegion(f *os.File, fileOff int64, addrOff, size uintptr) error {
	addr := m.ptr + addrOff
	if err := splitPlaceholder(addr, size); err != nil {
		return err
	}
	maxSize := uint64(fileOff) + uint64(size)
	h, err := windows.CreateFileMapping(windows.Handle(f.Fd()), nil,
		windows.PAGE_READWRITE, uint32(maxSize>>32), uint32(maxSize), nil)
	if h == 0 {
		m.restoreCommitted(addr, size)
		return err
	}
	if _, err := mapViewOfFile3(h, addr, uint64(fileOff), size,
		_MEM_REPLACE_PLACEHOLDER, windows.PAGE_READWRITE); err != nil {
		windows.CloseHandle(h)
		m.restoreCommitted(addr, size)
		return err
	}
	// The view keeps the section alive; the handle is no longer needed.
	windows.CloseHandle(h)
	m.views = append(m.views, addrOff)
	return nil
}

// restoreCommitted turns a placeholder back into committed private memory.
// It must not fail: a range of the linear memory would otherwise be left
// PAGE_NOACCESS, and whatever wasm code owns it would fault on next touch.
// Callers rely on the invariant that MapFileRegion/UnmapFileRegion return
// with the address space intact.
func (m *Memory) restoreCommitted(addr, size uintptr) {
	if _, err := virtualAlloc2(addr, size,
		windows.MEM_RESERVE|windows.MEM_COMMIT|_MEM_REPLACE_PLACEHOLDER, windows.PAGE_READWRITE); err != nil {
		panic(err)
	}
}

// UnmapFileRegion releases a view created by MapFileRegion and restores
// committed private (zeroed) memory in its place.
func (m *Memory) UnmapFileRegion(addrOff, size uintptr) error {
	addr := m.ptr + addrOff
	if err := unmapViewOfFile2(addr, _MEM_PRESERVE_PLACEHOLDER); err != nil {
		// The view is still mapped; the address space is intact.
		return err
	}
	// The view is gone: record that before anything else,
	// so Close never unmaps this offset a second time.
	for i, off := range m.views {
		if off == addrOff {
			m.views = append(m.views[:i], m.views[i+1:]...)
			break
		}
	}
	m.restoreCommitted(addr, size)
	return nil
}

func (m *Memory) Close() error {
	if m.ptr == 0 {
		m.Buf = nil
		return nil
	}
	var err error
	if m.placeh {
		// Unmap views first, then release every allocation and the
		// remaining placeholders piecewise.
		for _, off := range m.views {
			unmapViewOfFile2(m.ptr+off, 0)
		}
		// Walk the region, releasing each allocation.
		addr := m.ptr
		end := m.ptr + uintptr(cap(m.Buf))
		for addr < end {
			var info windows.MemoryBasicInformation
			if e := windows.VirtualQuery(addr, &info, unsafe.Sizeof(info)); e != nil {
				// Cannot advance without RegionSize; the rest leaks.
				if err == nil {
					err = e
				}
				break
			}
			if info.State != _MEM_FREE {
				if e := windows.VirtualFree(info.AllocationBase, 0, windows.MEM_RELEASE); e != nil && err == nil {
					err = e
				}
			}
			next := info.BaseAddress + info.RegionSize
			if next <= addr {
				break
			}
			addr = next
		}
	} else {
		err = windows.VirtualFree(m.ptr, 0, windows.MEM_RELEASE)
	}
	m.Buf = nil
	m.com = 0
	m.ptr = 0
	m.views = nil
	return err
}
