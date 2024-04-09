//go:build unix

package util

import (
	"math"

	"github.com/tetratelabs/wazero/experimental"
	"golang.org/x/sys/unix"
)

func mmappedAllocator(min, cap, max uint64) experimental.MemoryBuffer {
	if max > math.MaxInt {
		// This ensures int(max) overflows to a negative value,
		// and unix.Mmap returns EINVAL.
		max = math.MaxUint64
	}
	// Reserve the full max bytes of address space, to ensure we don't need to move it.
	// A protected, private, anonymous mapping should not commit memory.
	b, err := unix.Mmap(-1, 0, int(max), unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		panic(err)
	}
	// Commit the initial cap bytes of memory.
	err = unix.Mprotect(b[:cap], unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		unix.Munmap(b)
		panic(err)
	}
	return &mmappedBuffer{
		buf: b[:cap],
		cur: min,
	}
}

// The slice covers the entire mmapped memory:
//   - len(buf) is the already committed memory,
//   - cap(buf) is the reserved address space,
//   - cur is the already requested size.
type mmappedBuffer struct {
	buf []byte
	cur uint64
}

func (m *mmappedBuffer) Buffer() []byte {
	// Limit capacity because bytes beyond len(m.buf)
	// have not yet been committed.
	return m.buf[:m.cur:len(m.buf)]
}

func (m *mmappedBuffer) Grow(size uint64) []byte {
	// Commit additional memory up to size bytes.
	err := unix.Mprotect(m.buf[len(m.buf):size], unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		panic(err)
	}
	m.buf = m.buf[:size]
	m.cur = size
	return m.Buffer()
}

func (m *mmappedBuffer) Free() {
	err := unix.Munmap(m.buf)
	if err != nil {
		panic(err)
	}
	m.buf = nil
}
