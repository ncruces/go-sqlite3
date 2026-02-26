package util

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/ncruces/go-sqlite3/internal/sqlite3_wasm"
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

func View(mod *sqlite3_wasm.Module, ptr Ptr_t, size int64) []byte {
	if ptr == 0 {
		panic(NilErr)
	}
	return mod.Data[ptr:][:size:size]
}

func Read[T i8](mod *sqlite3_wasm.Module, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	return T(mod.Data[ptr])
}

func Write[T i8](mod *sqlite3_wasm.Module, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	mod.Data[ptr] = uint8(v)
}

func Read32[T i32](mod *sqlite3_wasm.Module, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	return T(binary.LittleEndian.Uint32(mod.Data[ptr:]))
}

func Write32[T i32](mod *sqlite3_wasm.Module, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	binary.LittleEndian.PutUint32(mod.Data[ptr:], uint32(v))
}

func Read64[T i64](mod *sqlite3_wasm.Module, ptr Ptr_t) T {
	if ptr == 0 {
		panic(NilErr)
	}
	return T(binary.LittleEndian.Uint64(mod.Data[ptr:]))
}

func Write64[T i64](mod *sqlite3_wasm.Module, ptr Ptr_t, v T) {
	if ptr == 0 {
		panic(NilErr)
	}
	binary.LittleEndian.PutUint64(mod.Data[ptr:], uint64(v))
}

func ReadFloat64(mod *sqlite3_wasm.Module, ptr Ptr_t) float64 {
	return math.Float64frombits(Read64[uint64](mod, ptr))
}

func WriteFloat64(mod *sqlite3_wasm.Module, ptr Ptr_t, v float64) {
	Write64(mod, ptr, math.Float64bits(v))
}

func ReadBool(mod *sqlite3_wasm.Module, ptr Ptr_t) bool {
	return Read32[int32](mod, ptr) != 0
}

func WriteBool(mod *sqlite3_wasm.Module, ptr Ptr_t, v bool) {
	var i int32
	if v {
		i = 1
	}
	Write32(mod, ptr, i)
}

func ReadString(mod *sqlite3_wasm.Module, ptr Ptr_t, maxlen int64) string {
	if ptr == 0 {
		panic(NilErr)
	}
	if maxlen <= 0 {
		return ""
	}
	buf := mod.Data[ptr:]
	if int64(len(buf)) > maxlen {
		buf = buf[:maxlen]
	}
	if i := bytes.IndexByte(buf, 0); i >= 0 {
		return string(buf[:i])
	}
	panic(NoNulErr)
}

func WriteBytes(mod *sqlite3_wasm.Module, ptr Ptr_t, b []byte) {
	buf := View(mod, ptr, int64(len(b)))
	copy(buf, b)
}

func WriteString(mod *sqlite3_wasm.Module, ptr Ptr_t, s string) {
	buf := View(mod, ptr, int64(len(s))+1)
	buf[len(s)] = 0
	copy(buf, s)
}
