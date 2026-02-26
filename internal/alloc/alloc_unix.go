//go:build unix

package alloc

import (
	"math"

	"golang.org/x/sys/unix"
)

// The slice covers the entire mmapped memory:
//   - len(buf) is the already committed memory,
//   - cap(buf) is the reserved address space.
type Memory struct {
	buf []byte
	Max int32
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
	rnd := uint64(unix.Getpagesize() - 1)
	res := (max + rnd) &^ rnd

	if res > math.MaxInt {
		// This ensures int(res) overflows to a negative value,
		// and unix.Mmap returns EINVAL.
		res = math.MaxUint64
	}

	// Reserve res bytes of address space, to ensure we won't need to move it.
	// A protected, private, anonymous mapping should not commit memory.
	b, err := unix.Mmap(-1, 0, int(res), unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		panic(err)
	}
	m.buf = b[:0]
}

func (m *Memory) reallocate(size uint64) []byte {
	com := uint64(len(m.buf))
	res := uint64(cap(m.buf))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(unix.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		err := unix.Mprotect(m.buf[com:new], unix.PROT_READ|unix.PROT_WRITE)
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
	err := unix.Munmap(m.buf[:cap(m.buf)])
	if err != nil {
		panic(err)
	}
	m.buf = nil
}
