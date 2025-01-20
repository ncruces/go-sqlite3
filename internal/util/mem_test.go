package util

import (
	"math"
	"testing"

	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

func TestView_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	View(mock, 0, 8)
	t.Error("want panic")
}

func TestView_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	View(mock, wazerotest.PageSize-2, 8)
	t.Error("want panic")
}

func TestView_overflow(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	View(mock, 1, math.MaxInt64)
	t.Error("want panic")
}

func TestReadUint8_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Read[byte](mock, 0)
	t.Error("want panic")
}

func TestReadUint8_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Read[byte](mock, wazerotest.PageSize)
	t.Error("want panic")
}

func TestReadUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Read32[uint32](mock, 0)
	t.Error("want panic")
}

func TestReadUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Read32[uint32](mock, wazerotest.PageSize-2)
	t.Error("want panic")
}

func TestReadUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Read64[uint64](mock, 0)
	t.Error("want panic")
}

func TestReadUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Read64[uint64](mock, wazerotest.PageSize-2)
	t.Error("want panic")
}

func TestWriteUint8_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Write[byte](mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint8_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Write[byte](mock, wazerotest.PageSize, 1)
	t.Error("want panic")
}

func TestWriteUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Write32[uint32](mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Write32[uint32](mock, wazerotest.PageSize-2, 1)
	t.Error("want panic")
}

func TestWriteUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Write64[uint64](mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	Write64[uint64](mock, wazerotest.PageSize-2, 1)
	t.Error("want panic")
}

func TestReadString_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	ReadString(mock, wazerotest.PageSize+2, math.MaxUint32)
	t.Error("want panic")
}
