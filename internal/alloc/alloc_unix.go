//go:build unix

package alloc

import (
	"math"

	"golang.org/x/sys/unix"
)

// The mem slice covers the entire mapped memory:
//   - len(mem) is the already committed memory,
//   - cap(mem) is the reserved address space.
type Memory struct {
	Max int32
	Buf []byte
	mem []byte
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
	m.mem = b[:0]
}

func (m *Memory) reallocate(size uint64) {
	com := uint64(len(m.mem))
	res := uint64(cap(m.mem))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(unix.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		err := unix.Mprotect(m.mem[com:new], unix.PROT_READ|unix.PROT_WRITE)
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
	err := unix.Munmap(m.mem[:cap(m.mem)])
	if err != nil {
		panic(err)
	}
	m.mem = nil
}
