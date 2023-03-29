package util

import (
	"math"
	"testing"
)

func TestView_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	View(mock, 0, 8)
	t.Error("want panic")
}

func TestView_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	View(mock, 126, 8)
	t.Error("want panic")
}

func TestView_overflow(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	View(mock, 1, math.MaxInt64)
	t.Error("want panic")
}

func TestReadUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	ReadUint32(mock, 0)
	t.Error("want panic")
}

func TestReadUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	ReadUint32(mock, 126)
	t.Error("want panic")
}

func TestReadUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	ReadUint64(mock, 0)
	t.Error("want panic")
}

func TestReadUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	ReadUint64(mock, 126)
	t.Error("want panic")
}

func TestWriteUint32_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	WriteUint32(mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint32_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	WriteUint32(mock, 126, 1)
	t.Error("want panic")
}

func TestWriteUint64_nil(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	WriteUint64(mock, 0, 1)
	t.Error("want panic")
}

func TestWriteUint64_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	WriteUint64(mock, 126, 1)
	t.Error("want panic")
}

func TestReadString_range(t *testing.T) {
	defer func() { _ = recover() }()
	mock := NewMockModule(128)
	ReadString(mock, 130, math.MaxUint32)
	t.Error("want panic")
}
