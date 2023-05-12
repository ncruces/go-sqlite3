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

func TestReadUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	ReadUint32(mock, 0)
	t.Error("want panic")
}

func TestReadUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	ReadUint32(mock, wazerotest.PageSize-2)
	t.Error("want panic")
}

func TestReadUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	ReadUint64(mock, 0)
	t.Error("want panic")
}

func TestReadUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	ReadUint64(mock, wazerotest.PageSize-2)
	t.Error("want panic")
}

func TestWriteUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	WriteUint32(mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	WriteUint32(mock, wazerotest.PageSize-2, 1)
	t.Error("want panic")
}

func TestWriteUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	WriteUint64(mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	WriteUint64(mock, wazerotest.PageSize-2, 1)
	t.Error("want panic")
}

func TestReadString_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := wazerotest.NewModule(wazerotest.NewFixedMemory(wazerotest.PageSize))
	ReadString(mock, wazerotest.PageSize+2, math.MaxUint32)
	t.Error("want panic")
}
