package sqlite3

import (
	"math"
	"testing"
)

func Test_memory_view_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.view(0, 8)
	t.Error("want panic")
}

func Test_memory_view_range(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.view(126, 8)
	t.Error("want panic")
}

func Test_memory_view_overflow(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.view(1, math.MaxInt64)
	t.Error("want panic")
}

func Test_memory_readUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.readUint32(0)
	t.Error("want panic")
}

func Test_memory_readUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.readUint32(126)
	t.Error("want panic")
}

func Test_memory_readUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.readUint64(0)
	t.Error("want panic")
}

func Test_memory_readUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.readUint64(126)
	t.Error("want panic")
}

func Test_memory_writeUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.writeUint32(0, 1)
	t.Error("want panic")
}

func Test_memory_writeUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.writeUint32(126, 1)
	t.Error("want panic")
}

func Test_memory_writeUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.writeUint64(0, 1)
	t.Error("want panic")
}

func Test_memory_writeUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.writeUint64(126, 1)
	t.Error("want panic")
}

func Test_memory_readString_range(t *testing.T) {
	defer func() { _ = recover() }()
	mem := newMemory(128)
	mem.readString(130, math.MaxUint32)
	t.Error("want panic")
}
