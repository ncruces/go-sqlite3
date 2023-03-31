package util

import (
	"math"
	"testing"
)

func Test_mockMemory_byte(t *testing.T) {
	const want byte = 98
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadByte(128)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteByte(128, 0)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteByte(0, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().ReadByte(0)
	if !ok {
		t.Error("want ok")
	}
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func Test_mockMemory_uint16(t *testing.T) {
	const want uint16 = 9876
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadUint16Le(128)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteUint16Le(128, 0)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteUint16Le(0, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().ReadUint16Le(0)
	if !ok {
		t.Error("want ok")
	}
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func Test_mockMemory_uint32(t *testing.T) {
	const want uint32 = 987654321
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadUint32Le(128)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteUint32Le(128, 0)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteUint32Le(0, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().ReadUint32Le(0)
	if !ok {
		t.Error("want ok")
	}
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func Test_mockMemory_uint64(t *testing.T) {
	const want uint64 = 9876543210
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadUint64Le(128)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteUint64Le(128, 0)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteUint64Le(0, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().ReadUint64Le(0)
	if !ok {
		t.Error("want ok")
	}
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func Test_mockMemory_float32(t *testing.T) {
	const want float32 = math.Pi
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadFloat32Le(128)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteFloat32Le(128, 0)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteFloat32Le(0, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().ReadFloat32Le(0)
	if !ok {
		t.Error("want ok")
	}
	if got != want {
		t.Errorf("got %f, want %f", got, want)
	}
}

func Test_mockMemory_float64(t *testing.T) {
	const want float64 = math.Pi
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadFloat64Le(128)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteFloat64Le(128, 0)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteFloat64Le(0, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().ReadFloat64Le(0)
	if !ok {
		t.Error("want ok")
	}
	if got != want {
		t.Errorf("got %f, want %f", got, want)
	}
}

func Test_mockMemory_bytes(t *testing.T) {
	const want string = "\xca\xfe\xba\xbe"
	mock := NewMockModule(128)

	_, ok := mock.Memory().Read(128, uint32(len(want)))
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().Write(128, []byte(want))
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().WriteString(128, want)
	if ok {
		t.Error("want error")
	}

	ok = mock.Memory().Write(0, []byte(want))
	if !ok {
		t.Error("want ok")
	}

	got, ok := mock.Memory().Read(0, uint32(len(want)))
	if !ok {
		t.Error("want ok")
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}

	ok = mock.Memory().WriteString(64, want)
	if !ok {
		t.Error("want ok")
	}

	got, ok = mock.Memory().Read(64, uint32(len(want)))
	if !ok {
		t.Error("want ok")
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_mockMemory_grow(t *testing.T) {
	mock := NewMockModule(128)

	_, ok := mock.Memory().ReadByte(65536)
	if ok {
		t.Error("want error")
	}

	got, ok := mock.Memory().Grow(1)
	if !ok {
		t.Error("want ok")
	}
	if got != 1 {
		t.Errorf("got %d, want 1", got)
	}

	_, ok = mock.Memory().ReadByte(65536)
	if !ok {
		t.Error("want ok")
	}
}
