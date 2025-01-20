package util

import (
	"bytes"
	"math"

	"github.com/tetratelabs/wazero/api"
)

const (
	PtrLen = 4
	IntLen = 4
)

type (
	Ptr_t uint32
	Res_t int32
)

func View(mod api.Module, ptr Ptr_t, size uint64) []byte {
	if ptr == 0 {
		panic(NilErr)
	}
	if size > math.MaxUint32 {
		panic(RangeErr)
	}
	if size == 0 {
		return nil
	}
	buf, ok := mod.Memory().Read(uint32(ptr), uint32(size))
	if !ok {
		panic(RangeErr)
	}
	return buf
}

func Read[T i8](mod api.Module, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	v, ok := mod.Memory().ReadByte(uint32(ptr))
	if !ok {
		panic(RangeErr)
	}
	return T(v)
}

func Write[T i8](mod api.Module, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	ok := mod.Memory().WriteByte(uint32(ptr), uint8(v))
	if !ok {
		panic(RangeErr)
	}
}

func Read32[T i32](mod api.Module, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	v, ok := mod.Memory().ReadUint32Le(uint32(ptr))
	if !ok {
		panic(RangeErr)
	}
	return T(v)
}

func Write32[T i32](mod api.Module, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	ok := mod.Memory().WriteUint32Le(uint32(ptr), uint32(v))
	if !ok {
		panic(RangeErr)
	}
}

func Read64[T i64](mod api.Module, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	v, ok := mod.Memory().ReadUint64Le(uint32(ptr))
	if !ok {
		panic(RangeErr)
	}
	return T(v)
}

func Write64[T i64](mod api.Module, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	ok := mod.Memory().WriteUint64Le(uint32(ptr), uint64(v))
	if !ok {
		panic(RangeErr)
	}
}

func ReadFloat64(mod api.Module, ptr Ptr_t) float64 {
	return math.Float64frombits(Read64[uint64](mod, ptr))
}

func WriteFloat64(mod api.Module, ptr Ptr_t, v float64) {
	Write64(mod, ptr, math.Float64bits(v))
}

func ReadString(mod api.Module, ptr Ptr_t, maxlen uint32) string {
	if ptr == 0 {
		panic(NilErr)
	}
	switch maxlen {
	case 0:
		return ""
	case math.MaxUint32:
		// avoid overflow
	default:
		maxlen = maxlen + 1
	}
	mem := mod.Memory()
	buf, ok := mem.Read(uint32(ptr), maxlen)
	if !ok {
		buf, ok = mem.Read(uint32(ptr), mem.Size()-uint32(ptr))
		if !ok {
			panic(RangeErr)
		}
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic(NoNulErr)
	} else {
		return string(buf[:i])
	}
}

func WriteBytes(mod api.Module, ptr Ptr_t, b []byte) {
	buf := View(mod, ptr, uint64(len(b)))
	copy(buf, b)
}

func WriteString(mod api.Module, ptr Ptr_t, s string) {
	buf := View(mod, ptr, uint64(len(s)+1))
	buf[len(s)] = 0
	copy(buf, s)
}
