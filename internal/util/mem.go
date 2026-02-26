package util

import (
	"bytes"
	"encoding/binary"
	"math"
)

const (
	PtrLen = 4
	IntLen = 4
)

type (
	i8  = interface{ ~int8 | ~uint8 }
	i32 = interface{ ~int32 | ~uint32 }
	i64 = interface{ ~int64 | ~uint64 }

	Ptr_t uint32
	Res_t int32
)

type Memory interface{ Data() *[]byte }

func View(mem Memory, ptr Ptr_t, size int64) []byte {
	if ptr == 0 {
		panic(NilErr)
	}
	return (*mem.Data())[ptr:][:size:size]
}

func Read[T i8](mem Memory, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	return T((*mem.Data())[ptr])
}

func Write[T i8](mem Memory, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	(*mem.Data())[ptr] = uint8(v)
}

func Read32[T i32](mem Memory, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	return T(binary.LittleEndian.Uint32((*mem.Data())[ptr:]))
}

func Write32[T i32](mem Memory, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	binary.LittleEndian.PutUint32((*mem.Data())[ptr:], uint32(v))
}

func Read64[T i64](mem Memory, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	return T(binary.LittleEndian.Uint64((*mem.Data())[ptr:]))
}

func Write64[T i64](mem Memory, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	binary.LittleEndian.PutUint64((*mem.Data())[ptr:], uint64(v))
}

func ReadFloat64(mem Memory, ptr Ptr_t) float64 {
	return math.Float64frombits(Read64[uint64](mem, ptr))
}

func WriteFloat64(mem Memory, ptr Ptr_t, v float64) {
	Write64(mem, ptr, math.Float64bits(v))
}

func ReadBool(mem Memory, ptr Ptr_t) bool {
	return Read32[int32](mem, ptr) != 0
}

func WriteBool(mem Memory, ptr Ptr_t, v bool) {
	var i int32
	if v {
		i = 1
	}
	Write32(mem, ptr, i)
}

func ReadString(mem Memory, ptr Ptr_t, maxlen int64) string {
	if ptr == 0 {
		panic(NilErr)
	}
	if maxlen <= 0 {
		return ""
	}
	buf := (*mem.Data())[ptr:]
	if int64(len(buf)) > maxlen {
		buf = buf[:maxlen]
	}
	if i := bytes.IndexByte(buf, 0); i >= 0 {
		return string(buf[:i])
	}
	panic(NoNulErr)
}

func WriteBytes(mem Memory, ptr Ptr_t, b []byte) {
	buf := View(mem, ptr, int64(len(b)))
	copy(buf, b)
}

func WriteString(mem Memory, ptr Ptr_t, s string) {
	buf := View(mem, ptr, int64(len(s))+1)
	buf[len(s)] = 0
	copy(buf, s)
}
