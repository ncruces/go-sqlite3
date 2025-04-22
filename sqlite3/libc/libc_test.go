package libc

import (
	"context"
	_ "embed"
	"os"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed libc.wasm
var binary []byte

const (
	size = 1024 * 1024 * 4
	ptr1 = 1024 * 1024
	ptr2 = ptr1 + size
)

var (
	memory  []byte
	module  api.Module
	memset  api.Function
	memcpy  api.Function
	memchr  api.Function
	memcmp  api.Function
	strlen  api.Function
	strchr  api.Function
	strcmp  api.Function
	strspn  api.Function
	strncmp api.Function
	strcspn api.Function
	stack   [8]uint64
)

func call(fn api.Function, arg ...uint64) uint32 {
	copy(stack[:], arg)
	fn.CallWithStack(context.Background(), stack[:])
	return uint32(stack[0])
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	runtime := wazero.NewRuntime(ctx)
	mod, err := runtime.Instantiate(ctx, binary)
	if err != nil {
		panic(err)
	}

	module = mod
	memset = mod.ExportedFunction("memset")
	memcpy = mod.ExportedFunction("memcpy")
	memchr = mod.ExportedFunction("memchr")
	memcmp = mod.ExportedFunction("memcmp")
	strlen = mod.ExportedFunction("strlen")
	strchr = mod.ExportedFunction("strchr")
	strcmp = mod.ExportedFunction("strcmp")
	strspn = mod.ExportedFunction("strspn")
	strncmp = mod.ExportedFunction("strncmp")
	strcspn = mod.ExportedFunction("strcspn")
	memory, _ = mod.Memory().Read(0, mod.Memory().Size())

	os.Exit(m.Run())
}

func Benchmark_memset(b *testing.B) {
	clear(memory)

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(memset, ptr1, 3, size)
	}
	b.StopTimer()

	for i, got := range memory[ptr1 : ptr1+size] {
		if got != 3 {
			b.Fatal(i, got)
		}
	}
}

func Benchmark_memcpy(b *testing.B) {
	clear(memory)
	call(memset, ptr2, 5, size)

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(memcpy, ptr1, ptr2, size)
	}
	b.StopTimer()

	for i, got := range memory[ptr1 : ptr1+size] {
		if got != 5 {
			b.Fatal(i, got)
		}
	}
}

func Benchmark_memchr(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size)
	call(memset, ptr1+size/2, 5, size/2)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(memchr, ptr1, 5, size)
	}
	b.StopTimer()

	if got := call(memchr, ptr1, 5, size); got != ptr1+size/2 {
		b.Fatal(got)
	}
	if got := call(memchr, ptr1, 5, size/2); got != 0 {
		b.Fatal(got, ptr1+size/2)
	}
}

func Benchmark_memcmp(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size)
	call(memset, ptr2, 7, size)
	call(memset, ptr2+size/2, 5, size/2)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(memcmp, ptr1, ptr2, size)
	}
	b.StopTimer()

	// ptr1 > ptr2
	if got := int32(call(memcmp, ptr1, ptr2, size)); got <= 0 {
		b.Fatal(got)
	}
	// ptr1[:size/2] == ptr2[:size/2]
	if got := int32(call(memcmp, ptr1, ptr2, size/2)); got != 0 {
		b.Fatal(got)
	}
}

func Benchmark_strlen(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 5, size-1)

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strlen, ptr1)
	}
	b.StopTimer()

	if got := int32(call(strlen, ptr1)); got != size-1 {
		b.Fatal(got)
	}
}

func Benchmark_strchr(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size-1)
	call(memset, ptr1+size/2, 5, size/2-1)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strchr, ptr1, 5)
	}
	b.StopTimer()

	if got := call(strchr, ptr1, 5); got != ptr1+size/2 {
		b.Fatal(got)
	}
}

func Benchmark_strcmp(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size-1)
	call(memset, ptr2, 7, size-1)
	call(memset, ptr2+size/2, 5, size/2-1)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strcmp, ptr1, ptr2, size)
	}
	b.StopTimer()

	// ptr1 > ptr2
	if got := int32(call(strcmp, ptr1, ptr2)); got <= 0 {
		b.Fatal(got)
	}
	// make ptr1 < ptr2
	memory[ptr1+size/2] = 0
	if got := int32(call(strcmp, ptr1, ptr2)); got >= 0 {
		b.Fatal(got)
	}
	memory[ptr2+size/2] = 0
	// make ptr1 == ptr2
	if got := int32(call(strcmp, ptr1, ptr2)); got != 0 {
		b.Fatal(got)
	}
}

func Benchmark_strncmp(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size-1)
	call(memset, ptr2, 7, size-1)
	call(memset, ptr2+size/2, 5, size/2-1)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strncmp, ptr1, ptr2, size-1)
	}
	b.StopTimer()

	// ptr1 > ptr2
	if got := int32(call(strncmp, ptr1, ptr2, size-1)); got <= 0 {
		b.Fatal(got)
	}
	// make ptr1 < ptr2
	memory[ptr1+size/2] = 0
	if got := int32(call(strncmp, ptr1, ptr2, size-1)); got >= 0 {
		b.Fatal(got)
	}
	// ptr1[:size/2] == ptr2[:size/2]
	if got := int32(call(strncmp, ptr1, ptr2, size/2-1)); got != 0 {
		b.Fatal(got)
	}
}

func Benchmark_strspn(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size-1)
	call(memset, ptr1+size/2, 5, size/2-1)
	memory[ptr2+0] = 3
	memory[ptr2+1] = 5
	memory[ptr2+2] = 7
	memory[ptr2+3] = 9

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strspn, ptr1, ptr2)
	}
	b.StopTimer()

	if got := int32(call(strspn, ptr1, ptr2)); got != size-1 {
		b.Fatal(got)
	}
	memory[ptr1+size/2] = 11
	if got := int32(call(strspn, ptr1, ptr2)); got != size/2 {
		b.Fatal(got)
	}
}

func Benchmark_strcspn(b *testing.B) {
	clear(memory)
	call(memset, ptr1, 7, size-1)
	call(memset, ptr1+size/2, 5, size/2-1)
	memory[ptr2+0] = 3
	memory[ptr2+1] = 9

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strcspn, ptr1, ptr2)
	}
	b.StopTimer()

	if got := int32(call(strcspn, ptr1, ptr2)); got != size-1 {
		b.Fatal(got)
	}
	memory[ptr1+size/2] = 3
	if got := int32(call(strcspn, ptr1, ptr2)); got != size/2 {
		b.Fatal(got)
	}
}
