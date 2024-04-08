//go:build (linux || darwin) && (amd64 || arm64)

package util

import (
	"math"
	"syscall"
)

type MmapedMemoryAllocator []byte

func (m *MmapedMemoryAllocator) Make(min, cap, max uint64) []byte {
	if max > math.MaxInt {
		// This ensures int(max) overflows to a negative value,
		// and syscall.Mmap returns EINVAL.
		max = math.MaxUint64
	}
	// Reserve the full max bytes of address space, to ensure we don't need to move it.
	// A protected, private, anonymous mapping should not commit memory.
	b, err := syscall.Mmap(-1, 0, int(max), syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		panic(err)
	}
	// Commit the initial min bytes of memory.
	err = syscall.Mprotect(b[:min], syscall.PROT_READ|syscall.PROT_WRITE)
	if err != nil {
		syscall.Munmap(b)
		panic(err)
	}
	b = b[:min]
	*m = b
	return b
}

func (m *MmapedMemoryAllocator) Grow(size uint64) []byte {
	b := *m
	// Commit additional memory up to size bytes.
	err := syscall.Mprotect(b[len(b):size], syscall.PROT_READ|syscall.PROT_WRITE)
	if err != nil {
		panic(err)
	}
	b = b[:size]
	*m = b
	return b
}

func (m *MmapedMemoryAllocator) Free() {
	err := syscall.Munmap(*m)
	if err != nil {
		panic(err)
	}
	*m = nil
}
