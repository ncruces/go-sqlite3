//go:build !sqlite3_nosys

package util

import (
	"context"
	"math"
	"reflect"
	"unsafe"

	"github.com/tetratelabs/wazero/experimental"
	"golang.org/x/sys/windows"
)

func withAllocator(ctx context.Context) context.Context {
	if math.MaxInt != math.MaxInt64 {
		return ctx
	}
	return experimental.WithMemoryAllocator(ctx,
		experimental.MemoryAllocatorFunc(newAllocator))
}

func newAllocator(cap, max uint64) experimental.LinearMemory {
	// Round up to the page size.
	rnd := uint64(windows.Getpagesize() - 1)
	max = (max + rnd) &^ rnd
	cap = (cap + rnd) &^ rnd

	if max > math.MaxInt {
		// This ensures int(max) overflows to a negative value,
		// and unix.Mmap returns EINVAL.
		max = math.MaxUint64
	}
	// Reserve max bytes of address space, to ensure we won't need to move it.
	// A protected, private, anonymous mapping should not commit memory.
	r, err := windows.VirtualAlloc(0, uintptr(max), windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		panic(err)
	}
	// Commit the initial cap bytes of memory.
	_, err = windows.VirtualAlloc(r, uintptr(cap), windows.MEM_COMMIT, windows.PAGE_READWRITE)
	if err != nil {
		windows.VirtualFree(r, 0, windows.MEM_RELEASE)
		panic(err)
	}
	mem := virtualMemory{addr: r}
	// SliceHeader, although deprecated, avoids a go vet warning.
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&mem.buf))
	sh.Len = int(cap) // Not a bug.
	sh.Cap = int(max) // Not a bug.
	sh.Data = r
	return &mem
}

// The slice covers the entire mmapped memory:
//   - len(buf) is the already committed memory,
//   - cap(buf) is the reserved address space.
type virtualMemory struct {
	buf  []byte
	addr uintptr
}

func (m *virtualMemory) Reallocate(size uint64) []byte {
	if com := uint64(len(m.buf)); com < size {
		// Round up to the page size.
		rnd := uint64(windows.Getpagesize() - 1)
		new := (size + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		_, err := windows.VirtualAlloc(m.addr, uintptr(new), windows.MEM_COMMIT, windows.PAGE_READWRITE)
		if err != nil {
			panic(err)
		}

		// Update committed memory.
		m.buf = m.buf[:new]
	}
	// Limit returned capacity because bytes beyond
	// len(m.buf) have not yet been committed.
	return m.buf[:size:len(m.buf)]
}

func (m *virtualMemory) Free() {
	err := windows.VirtualFree(m.addr, 0, windows.MEM_RELEASE)
	if err != nil {
		panic(err)
	}
	m.addr = 0
}
