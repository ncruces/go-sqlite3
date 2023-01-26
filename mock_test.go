package sqlite3

import (
	"context"
	"encoding/binary"
	"math"

	"github.com/tetratelabs/wazero/api"
)

func newMemory(size uint32) memory {
	mem := make(mockMemory, size)
	return memory{mockModule{&mem}}
}

type mockModule struct {
	memory api.Memory
}

func (m mockModule) Memory() api.Memory { return m.memory }
func (m mockModule) String() string     { return "mockModule" }
func (m mockModule) Name() string       { return "mockModule" }

func (m mockModule) ExportedGlobal(name string) api.Global                          { return nil }
func (m mockModule) ExportedMemory(name string) api.Memory                          { return nil }
func (m mockModule) ExportedFunction(name string) api.Function                      { return nil }
func (m mockModule) ExportedMemoryDefinitions() map[string]api.MemoryDefinition     { return nil }
func (m mockModule) ExportedFunctionDefinitions() map[string]api.FunctionDefinition { return nil }
func (m mockModule) CloseWithExitCode(ctx context.Context, exitCode uint32) error   { return nil }
func (m mockModule) Close(context.Context) error                                    { return nil }

type mockMemory []byte

func (m mockMemory) Definition() api.MemoryDefinition { return nil }

func (m mockMemory) Size() uint32 { return uint32(len(m)) }

func (m mockMemory) ReadByte(offset uint32) (byte, bool) {
	if offset >= m.Size() {
		return 0, false
	}
	return m[offset], true
}

func (m mockMemory) ReadUint16Le(offset uint32) (uint16, bool) {
	if !m.hasSize(offset, 2) {
		return 0, false
	}
	return binary.LittleEndian.Uint16(m[offset : offset+2]), true
}

func (m mockMemory) ReadUint32Le(offset uint32) (uint32, bool) {
	if !m.hasSize(offset, 4) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(m[offset : offset+4]), true
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
	return binary.LittleEndian.Uint64(m[offset : offset+8]), true
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
	return m[offset : offset+byteCount : offset+byteCount], true
}

func (m mockMemory) WriteByte(offset uint32, v byte) bool {
	if offset >= m.Size() {
		return false
	}
	m[offset] = v
	return true
}

func (m mockMemory) WriteUint16Le(offset uint32, v uint16) bool {
	if !m.hasSize(offset, 2) {
		return false
	}
	binary.LittleEndian.PutUint16(m[offset:], v)
	return true
}

func (m mockMemory) WriteUint32Le(offset, v uint32) bool {
	if !m.hasSize(offset, 4) {
		return false
	}
	binary.LittleEndian.PutUint32(m[offset:], v)
	return true
}

func (m mockMemory) WriteFloat32Le(offset uint32, v float32) bool {
	return m.WriteUint32Le(offset, math.Float32bits(v))
}

func (m mockMemory) WriteUint64Le(offset uint32, v uint64) bool {
	if !m.hasSize(offset, 8) {
		return false
	}
	binary.LittleEndian.PutUint64(m[offset:], v)
	return true
}

func (m mockMemory) WriteFloat64Le(offset uint32, v float64) bool {
	return m.WriteUint64Le(offset, math.Float64bits(v))
}

func (m mockMemory) Write(offset uint32, val []byte) bool {
	if !m.hasSize(offset, uint32(len(val))) {
		return false
	}
	copy(m[offset:], val)
	return true
}

func (m mockMemory) WriteString(offset uint32, val string) bool {
	if !m.hasSize(offset, uint32(len(val))) {
		return false
	}
	copy(m[offset:], val)
	return true
}

func (m *mockMemory) Grow(delta uint32) (result uint32, ok bool) {
	mem := append(*m, make([]byte, 65536)...)
	m = &mem
	return delta, true
}

func (m mockMemory) PageSize() (result uint32) {
	return uint32(len(m) / 65536)
}

func (m mockMemory) hasSize(offset uint32, byteCount uint32) bool {
	return uint64(offset)+uint64(byteCount) <= uint64(len(m))
}
