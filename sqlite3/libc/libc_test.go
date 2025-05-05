package libc

import (
	"context"
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed libc.wasm
var binary []byte

const (
	page = 64 * 1024
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
	strrchr api.Function
	strncmp api.Function
	strcspn api.Function
	stack   [8]uint64
)

func call(fn api.Function, arg ...uint64) uint64 {
	copy(stack[:], arg)
	err := fn.CallWithStack(context.Background(), stack[:])
	if err != nil {
		panic(err)
	}
	return stack[0]
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
	strrchr = mod.ExportedFunction("strrchr")
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
}

func Benchmark_memcpy(b *testing.B) {
	clear(memory)
	fill(memory[ptr2:ptr2+size], 5)

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(memcpy, ptr1, ptr2, size)
	}
}

func Benchmark_memchr(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size/2], 7)
	fill(memory[ptr1+size/2:ptr1+size], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(memchr, ptr1, 5, size)
	}
}

func Benchmark_memcmp(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size], 7)
	fill(memory[ptr2:ptr2+size/2], 7)
	fill(memory[ptr2+size/2:ptr2+size], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(memcmp, ptr1, ptr2, size)
	}
}

func Benchmark_strlen(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size-1], 5)

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strlen, ptr1)
	}
}

func Benchmark_strchr(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size/2], 7)
	fill(memory[ptr1+size/2:ptr1+size-1], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strchr, ptr1, 5)
	}
}

func Benchmark_strrchr(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size/2], 5)
	fill(memory[ptr1+size/2:ptr1+size-1], 7)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strrchr, ptr1, 5)
	}
}

func Benchmark_strcmp(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size-1], 7)
	fill(memory[ptr2:ptr2+size/2], 7)
	fill(memory[ptr2+size/2:ptr2+size-1], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strcmp, ptr1, ptr2, size)
	}
}

func Benchmark_strncmp(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size-1], 7)
	fill(memory[ptr2:ptr2+size/2], 7)
	fill(memory[ptr2+size/2:ptr2+size-1], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strncmp, ptr1, ptr2, size-1)
	}
}

func Benchmark_strspn(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size/2], 7)
	fill(memory[ptr1+size/2:ptr1+size-1], 5)
	memory[ptr2+0] = 3
	memory[ptr2+1] = 5
	memory[ptr2+2] = 7
	memory[ptr2+3] = 9

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strspn, ptr1, ptr2)
	}
}

func Benchmark_strcspn(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size/2], 7)
	fill(memory[ptr1+size/2:ptr1+size-1], 5)
	memory[ptr2+0] = 3
	memory[ptr2+1] = 9

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strcspn, ptr1, ptr2)
	}
}

func Test_memcmp(t *testing.T) {
	const s1 string = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\x14\xf4\x93\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x80\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x7f\xf3\x93\x01\x00\x01"
	const s2 string = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\xbc\x40\x96\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x7f\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x80\xf3\x93\x01\x00\x02"

	p1 := ptr1
	p2 := len(memory) - len(s2)

	clear(memory)
	copy(memory[p1:], s1)
	copy(memory[p2:], s2)

	for i := range len(s1) + 1 {
		for j := range len(s1) - i {
			want := strings.Compare(s1[i:i+j], s2[i:i+j])
			got := call(memcmp, uint64(p1+i), uint64(p2+i), uint64(j))
			if sign(int32(got)) != want {
				t.Errorf("strcmp(%d, %d, %d) = %d, want %d",
					ptr1+i, ptr2+i, j, int32(got), want)
			}
		}
	}
}

func Test_strcmp(t *testing.T) {
	const s1 string = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\x14\xf4\x93\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x80\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x7f\xf3\x93\x01\x00\x01"
	const s2 string = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\xbc\x40\x96\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x7f\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x80\xf3\x93\x01\x00\x02"

	p1 := ptr1
	p2 := len(memory) - len(s2) - 1

	clear(memory)
	copy(memory[p1:], s1)
	copy(memory[p2:], s2)

	for i := range len(s1) + 1 {
		want := strings.Compare(term(s1[i:]), term(s2[i:]))
		got := call(strcmp, uint64(p1+i), uint64(p2+i))
		if sign(int32(got)) != want {
			t.Errorf("strcmp(%d, %d) = %d, want %d",
				p1+i, ptr2+i, int32(got), want)
		}
	}
}

func Test_strncmp(t *testing.T) {
	const s1 string = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\x14\xf4\x93\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x80\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x7f\xf3\x93\x01\x00\x01"
	const s2 string = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\xbc\x40\x96\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x7f\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x80\xf3\x93\x01\x00\x02"

	p1 := ptr1
	p2 := len(memory) - len(s2) - 1

	clear(memory)
	copy(memory[p1:], s1)
	copy(memory[p2:], s2)

	for i := range len(s1) + 1 {
		for j := range len(s1) - i + 1 {
			want := strings.Compare(term(s1[i:i+j]), term(s2[i:i+j]))
			got := call(strncmp, uint64(p1+i), uint64(p2+i), uint64(j))
			if sign(int32(got)) != want {
				t.Errorf("strncmp(%d, %d, %d) = %d, want %d",
					ptr1+i, ptr2+i, j, int32(got), want)
			}
		}
	}
}

func Test_strlen(t *testing.T) {
	for length := range 64 {
		for alignment := range 24 {
			ptr := (page - 8) + alignment

			clear(memory[:2*page])
			fill(memory[ptr:ptr+length], 5)

			got := call(strlen, uint64(ptr))
			if uint32(got) != uint32(length) {
				t.Errorf("strlen(%d) = %d, want %d",
					ptr, uint32(got), uint32(length))
			}

			memory[ptr-1] = 5
			got = call(strlen, uint64(ptr))
			if uint32(got) != uint32(length) {
				t.Errorf("strlen(%d) = %d, want %d",
					ptr, uint32(got), uint32(length))
			}
		}

		clear(memory)
		ptr := len(memory) - length - 1
		fill(memory[ptr:ptr+length], 5)

		got := call(strlen, uint64(ptr))
		if uint32(got) != uint32(length) {
			t.Errorf("strlen(%d) = %d, want %d",
				ptr, uint32(got), uint32(length))
		}
	}
}

func Test_memchr(t *testing.T) {
	for length := range 64 {
		for pos := range length + 2 {
			for alignment := range 24 {
				ptr := (page - 8) + alignment
				want := 0
				if pos < length {
					want = ptr + pos
				}

				clear(memory[:2*page])
				fill(memory[ptr:ptr+max(pos, length)], 5)
				memory[ptr+pos] = 7

				got := call(memchr, uint64(ptr), 7, uint64(length))
				if uint32(got) != uint32(want) {
					t.Errorf("memchr(%d, %d, %d) = %d, want %d",
						ptr, 7, uint64(length), uint32(got), uint32(want))
				}
			}
		}

		clear(memory)
		ptr := len(memory) - length
		fill(memory[ptr:ptr+length], 5)
		memory[len(memory)-1] = 7

		want := len(memory) - 1
		if length == 0 {
			want = 0
		}

		got := call(memchr, uint64(ptr), 7, uint64(length))
		if uint32(got) != uint32(want) {
			t.Errorf("memchr(%d, %d, %d) = %d, want %d",
				ptr, 7, uint64(length), uint32(got), uint32(want))
		}
	}
}

func Test_strchr(t *testing.T) {
	for length := range 64 {
		for pos := range length + 2 {
			for alignment := range 24 {
				ptr := (page - 8) + alignment
				want := 0
				if pos < length {
					want = ptr + pos
				}

				clear(memory[:2*page])
				fill(memory[ptr:ptr+max(pos, length)], 5)
				memory[ptr+pos] = 7
				memory[ptr+pos+1] = 7
				memory[ptr+length] = 0

				got := call(strchr, uint64(ptr), 7)
				if uint32(got) != uint32(want) {
					t.Errorf("strchr(%d, %d) = %d, want %d",
						ptr, 7, uint32(got), uint32(want))
				}
			}
		}

		clear(memory)
		ptr := len(memory) - length
		fill(memory[ptr:ptr+length], 5)
		memory[len(memory)-1] = 7

		want := len(memory) - 1
		if length == 0 {
			continue
		}

		got := call(strchr, uint64(ptr), 7)
		if uint32(got) != uint32(want) {
			t.Errorf("strchr(%d, %d) = %d, want %d",
				ptr, 7, uint32(got), uint32(want))
		}
	}
}

func Test_strrchr(t *testing.T) {
	for length := range 64 {
		for pos := range length + 2 {
			for alignment := range 24 {
				ptr := (page - 8) + alignment
				want := 0
				if pos < length {
					want = ptr + pos
				} else if length > 0 {
					want = ptr
				}

				clear(memory[:2*page])
				fill(memory[ptr:ptr+max(pos, length)], 5)
				memory[ptr] = 7
				memory[ptr+pos] = 7
				memory[ptr+length] = 0

				got := call(strrchr, uint64(ptr), 7)
				if uint32(got) != uint32(want) {
					t.Errorf("strrchr(%d, %d) = %d, want %d",
						ptr, 7, uint32(got), uint32(want))
				}
			}
		}

		ptr := len(memory) - length
		want := len(memory) - 2
		if length <= 1 {
			continue
		}

		clear(memory)
		fill(memory[ptr:ptr+length], 5)
		memory[ptr] = 7
		memory[len(memory)-2] = 7
		memory[len(memory)-1] = 0

		got := call(strrchr, uint64(ptr), 7)
		if uint32(got) != uint32(want) {
			t.Errorf("strrchr(%d, %d) = %d, want %d",
				ptr, 7, uint32(got), uint32(want))
		}
	}
}

func Test_strspn(t *testing.T) {
	for length := range 64 {
		for pos := range length + 2 {
			for alignment := range 24 {
				ptr := (page - 8) + alignment
				want := min(pos, length)

				clear(memory[:2*page])
				fill(memory[ptr:ptr+max(pos, length)], 5)
				memory[ptr+pos] = 7
				memory[ptr+length] = 0
				memory[128] = 3
				memory[129] = 5

				got := call(strspn, uint64(ptr), 129)
				if uint32(got) != uint32(want) {
					t.Errorf("strspn(%d, %d) = %d, want %d",
						ptr, 129, uint32(got), uint32(want))
				}

				got = call(strspn, uint64(ptr), 128)
				if uint32(got) != uint32(want) {
					t.Errorf("strspn(%d, %d) = %d, want %d",
						ptr, 128, uint32(got), uint32(want))
				}
			}
		}

		ptr := len(memory) - length
		want := length - 1
		if length == 0 {
			continue
		}

		clear(memory)
		fill(memory[ptr:ptr+length], 5)
		memory[len(memory)-1] = 7
		memory[128] = 3
		memory[129] = 5

		got := call(strspn, uint64(ptr), 129)
		if uint32(got) != uint32(want) {
			t.Errorf("strspn(%d, %d) = %d, want %d",
				ptr, 129, uint32(got), uint32(want))
		}

		got = call(strspn, uint64(ptr), 128)
		if uint32(got) != uint32(want) {
			t.Errorf("strspn(%d, %d) = %d, want %d",
				ptr, 128, uint32(got), uint32(want))
		}
	}
}

func Test_strcspn(t *testing.T) {
	for length := range 64 {
		for pos := range length + 2 {
			for alignment := range 24 {
				ptr := (page - 8) + alignment
				want := min(pos, length)

				clear(memory[:2*page])
				fill(memory[ptr:ptr+max(pos, length)], 5)
				memory[ptr+pos] = 7
				memory[ptr+length] = 0
				memory[128] = 3
				memory[129] = 7

				got := call(strcspn, uint64(ptr), 129)
				if uint32(got) != uint32(want) {
					t.Errorf("strcspn(%d, %d) = %d, want %d",
						ptr, 129, uint32(got), uint32(want))
				}

				got = call(strcspn, uint64(ptr), 128)
				if uint32(got) != uint32(want) {
					t.Errorf("strcspn(%d, %d) = %d, want %d",
						ptr, 128, uint32(got), uint32(want))
				}
			}
		}

		ptr := len(memory) - length
		want := length - 1
		if length == 0 {
			continue
		}

		clear(memory)
		fill(memory[ptr:ptr+length], 5)
		memory[len(memory)-1] = 7
		memory[128] = 3
		memory[129] = 7

		got := call(strcspn, uint64(ptr), 129)
		if uint32(got) != uint32(want) {
			t.Errorf("strcspn(%d, %d) = %d, want %d",
				ptr, 129, uint32(got), uint32(want))
		}

		got = call(strcspn, uint64(ptr), 128)
		if uint32(got) != uint32(want) {
			t.Errorf("strcspn(%d, %d) = %d, want %d",
				ptr, 128, uint32(got), uint32(want))
		}
	}
}

func fill(s []byte, v byte) {
	for i := range s {
		s[i] = v
	}
}

func sign(x int32) int {
	switch {
	case x > 0:
		return +1
	case x < 0:
		return -1
	default:
		return 0
	}
}

func term(s string) string {
	if i := strings.IndexByte(s, 0); i >= 0 {
		return s[:i]
	}
	return s
}
