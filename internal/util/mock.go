package util

import (
	"encoding/binary"
	"math"

	"github.com/tetratelabs/wazero/api"
)

func NewMockModule(size uint32) api.Module {
	mem := mockMemory{buf: make([]byte, size)}
	return mockModule{&mem, nil}
}

type mockModule struct {
	memory api.Memory
	api.Module
}

func (m mockModule) Memory() api.Memory { return m.memory }
func (m mockModule) String() string     { return "mockModule" }
func (m mockModule) Name() string       { return "mockModule" }

type mockMemory struct {
	buf []byte
	api.Memory
}

func (m mockMemory) Definition() api.MemoryDefinition { return nil }

func (m mockMemory) Size() uint32 { return uint32(len(m.buf)) }

func (m mockMemory) ReadByte(offset uint32) (byte, bool) {
	if offset >= m.Size() {
		return 0, false
	}
	return m.buf[offset], true
}

func (m mockMemory) ReadUint16Le(offset uint32) (uint16, bool) {
	if !m.hasSize(offset, 2) {
		return 0, false
	}
	return binary.LittleEndian.Uint16(m.buf[offset : offset+2]), true
}

func (m mockMemory) ReadUint32Le(offset uint32) (uint32, bool) {
	if !m.hasSize(offset, 4) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(m.buf[offset : offset+4]), true
}

func (m mockMemory) ReadFloat32Le(offset uint32) (float32, bool) {
	v, ok := m.ReadUint32Le(offset)
	if !ok {
		return 0, false
	}
	return math.Float32frombits(v), true
}

func (m mockMemory) ReadUint64Le(offset uint32) (uint64, bool) {
	if !m.hasSize(offset, 8) {
		return 0, false
	}
	return binary.LittleEndian.Uint64(m.buf[offset : offset+8]), true
}

func (m mockMemory) ReadFloat64Le(offset uint32) (float64, bool) {
	v, ok := m.ReadUint64Le(offset)
	if !ok {
		return 0, false
	}
	return math.Float64frombits(v), true
}

func (m mockMemory) Read(offset, byteCount uint32) ([]byte, bool) {
	if !m.hasSize(offset, byteCount) {
		return nil, false
	}
	return m.buf[offset : offset+byteCount : offset+byteCount], true
}

func (m mockMemory) WriteByte(offset uint32, v byte) bool {
	if offset >= m.Size() {
		return false
	}
	m.buf[offset] = v
	return true
}

func (m mockMemory) WriteUint16Le(offset uint32, v uint16) bool {
	if !m.hasSize(offset, 2) {
		return false
	}
	binary.LittleEndian.PutUint16(m.buf[offset:], v)
	return true
}

func (m mockMemory) WriteUint32Le(offset, v uint32) bool {
	if !m.hasSize(offset, 4) {
		return false
	}
	binary.LittleEndian.PutUint32(m.buf[offset:], v)
	return true
}

func (m mockMemory) WriteFloat32Le(offset uint32, v float32) bool {
	return m.WriteUint32Le(offset, math.Float32bits(v))
}

func (m mockMemory) WriteUint64Le(offset uint32, v uint64) bool {
	if !m.hasSize(offset, 8) {
		return false
	}
	binary.LittleEndian.PutUint64(m.buf[offset:], v)
	return true
}

func (m mockMemory) WriteFloat64Le(offset uint32, v float64) bool {
	return m.WriteUint64Le(offset, math.Float64bits(v))
}

func (m mockMemory) Write(offset uint32, val []byte) bool {
	if !m.hasSize(offset, uint32(len(val))) {
		return false
	}
	copy(m.buf[offset:], val)
	return true
}

func (m mockMemory) WriteString(offset uint32, val string) bool {
	if !m.hasSize(offset, uint32(len(val))) {
		return false
	}
	copy(m.buf[offset:], val)
	return true
}

func (m *mockMemory) Grow(delta uint32) (result uint32, ok bool) {
	prev := (len(m.buf) + 65535) / 65536
	m.buf = append(m.buf, make([]byte, 65536*delta)...)
	return uint32(prev), true
}

func (m mockMemory) hasSize(offset uint32, byteCount uint32) bool {
	return uint64(offset)+uint64(byteCount) <= uint64(len(m.buf))
}
