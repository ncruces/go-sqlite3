package sqlite3

import (
	"bytes"
	"math"

	"github.com/tetratelabs/wazero/api"
)

type memory struct {
	mod api.Module
}

func (m memory) size() uint32 {
	return m.mod.Memory().Size()
}

func (m memory) read(offset, byteCount uint32) ([]byte, bool) {
	if offset == 0 {
		panic(nilErr)
	}
	return m.mod.Memory().Read(offset, byteCount)
}

func (m memory) mustRead(offset, byteCount uint32) []byte {
	buf, ok := m.read(offset, byteCount)
	if !ok {
		panic(rangeErr)
	}
	return buf
}

func (m memory) readUint32(offset uint32) uint32 {
	if offset == 0 {
		panic(nilErr)
	}
	v, ok := m.mod.Memory().ReadUint32Le(offset)
	if !ok {
		panic(rangeErr)
	}
	return v
}

func (m memory) writeUint32(offset, v uint32) {
	if offset == 0 {
		panic(nilErr)
	}
	ok := m.mod.Memory().WriteUint32Le(offset, v)
	if !ok {
		panic(rangeErr)
	}
}

func (m memory) readUint64(offset uint32) uint64 {
	if offset == 0 {
		panic(nilErr)
	}
	v, ok := m.mod.Memory().ReadUint64Le(offset)
	if !ok {
		panic(rangeErr)
	}
	return v
}

func (m memory) writeUint64(offset uint32, v uint64) {
	if offset == 0 {
		panic(nilErr)
	}
	ok := m.mod.Memory().WriteUint64Le(offset, v)
	if !ok {
		panic(rangeErr)
	}
}

func (m memory) readFloat64(offset uint32) float64 {
	return math.Float64frombits(m.readUint64(offset))
}

func (m memory) writeFloat64(offset uint32, v float64) {
	m.writeUint64(offset, math.Float64bits(v))
}

func (m memory) readString(ptr, maxlen uint32) string {
	switch maxlen {
	case 0:
		return ""
	case math.MaxUint32:
		//
	default:
		maxlen = maxlen + 1
	}
	buf, ok := m.read(ptr, maxlen)
	if !ok {
		buf = m.mustRead(ptr, m.size()-ptr)
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic(noNulErr)
	} else {
		return string(buf[:i])
	}
}
