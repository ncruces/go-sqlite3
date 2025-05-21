package libc

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

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
	memory      []byte
	module      api.Module
	memset      api.Function
	memcpy      api.Function
	memchr      api.Function
	memcmp      api.Function
	memmem      api.Function
	strlen      api.Function
	strchr      api.Function
	strcmp      api.Function
	strstr      api.Function
	strspn      api.Function
	strrchr     api.Function
	strncmp     api.Function
	strcspn     api.Function
	strcasecmp  api.Function
	strcasestr  api.Function
	strncasecmp api.Function
	stack       [8]uint64
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
	memmem = mod.ExportedFunction("memmem")
	strlen = mod.ExportedFunction("strlen")
	strchr = mod.ExportedFunction("strchr")
	strcmp = mod.ExportedFunction("strcmp")
	strstr = mod.ExportedFunction("strstr")
	strspn = mod.ExportedFunction("strspn")
	strrchr = mod.ExportedFunction("strrchr")
	strncmp = mod.ExportedFunction("strncmp")
	strcspn = mod.ExportedFunction("strcspn")
	strcasecmp = mod.ExportedFunction("strcasecmp")
	strcasestr = mod.ExportedFunction("strcasestr")
	strncasecmp = mod.ExportedFunction("strncasecmp")
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

func Benchmark_strlen(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size-1], 5)

	b.SetBytes(size)
	b.ResetTimer()
	for range b.N {
		call(strlen, ptr1)
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

func Benchmark_strcasecmp(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size-1], 7)
	fill(memory[ptr2:ptr2+size/2], 7)
	fill(memory[ptr2+size/2:ptr2+size-1], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strcasecmp, ptr1, ptr2, size)
	}
}

func Benchmark_strncasecmp(b *testing.B) {
	clear(memory)
	fill(memory[ptr1:ptr1+size-1], 7)
	fill(memory[ptr2:ptr2+size/2], 7)
	fill(memory[ptr2+size/2:ptr2+size-1], 5)

	b.SetBytes(size/2 + 1)
	b.ResetTimer()
	for range b.N {
		call(strncasecmp, ptr1, ptr2, size-1)
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

//go:embed string.h
var source string

func Benchmark_memmem(b *testing.B) {
	needle := "memcpy(dest, src, slen)"

	clear(memory)
	copy(memory[ptr1:], source)
	copy(memory[ptr2:], needle)

	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for range b.N {
		call(memmem, ptr1, uint64(len(source)), ptr2, uint64(len(needle)))
	}
}

func Benchmark_strstr(b *testing.B) {
	needle := "memcpy(dest, src, slen)"

	clear(memory)
	copy(memory[ptr1:], source)
	copy(memory[ptr2:], needle)

	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for range b.N {
		call(strstr, ptr1, ptr2)
	}
}

func Benchmark_strcasestr(b *testing.B) {
	needle := "MEMCPY(dest, src, slen)"

	clear(memory)
	copy(memory[ptr1:], source)
	copy(memory[ptr2:], needle)

	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for range b.N {
		call(strcasestr, ptr1, ptr2)
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

const (
	compareTest1 = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\x14\xf4\x93\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x80\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x7f\xf3\x93\x01\x00\x01"
	compareTest2 = "" +
		"\x94\x63\x8f\x01\x74\x63\x8f\x01\x54\x63\x8f\x01\x34\x63\x8f\x01" +
		"\xb4\xf2\x93\x01\x94\xf2\x93\x01\x54\xf1\x93\x01\x34\xf1\x93\x01" +
		"\x14\xf1\x93\x01\x14\xf2\x93\x01\x34\xf2\x93\x01\x54\xf2\x93\x01" +
		"\x74\xf2\x93\x01\x74\xf1\x93\x01\xd4\xf2\x93\x01\x94\xf1\x93\x01" +
		"\xb4\xf1\x93\x01\xd4\xf1\x93\x01\xf4\xf1\x93\x01\xf4\xf2\x93\x01" +
		"\xbc\x40\x96\x01\xf4\xf3\x93\x01\xd4\xf3\x93\x01\xb4\xf3\x93\x01" +
		"\x94\xf3\x93\x01\x74\x7f\x93\x01\x54\xf3\x93\x01\x34\xf3\x93\x01" +
		"\x80\xf3\x93\x01\x00\x02"
)

func Test_memcmp(t *testing.T) {
	const s1 = compareTest1
	const s2 = compareTest2

	ptr2 := len(memory) - len(s2)

	clear(memory)
	copy(memory[ptr1:], s1)
	copy(memory[ptr2:], s2)

	for i := range len(s1) + 1 {
		for j := range len(s1) - i {
			want := strings.Compare(s1[i:i+j], s2[i:i+j])
			got := call(memcmp, uint64(ptr1+i), uint64(ptr2+i), uint64(j))
			if sign(int32(got)) != want {
				t.Errorf("strcmp(%d, %d, %d) = %d, want %d",
					ptr1+i, ptr2+i, j, int32(got), want)
			}
		}
	}
}

func Test_strcmp(t *testing.T) {
	const s1 = compareTest1
	const s2 = compareTest2

	ptr2 := len(memory) - len(s2) - 1

	clear(memory)
	copy(memory[ptr1:], s1)
	copy(memory[ptr2:], s2)

	for i := range len(s1) + 1 {
		want := strings.Compare(term(s1[i:]), term(s2[i:]))
		got := call(strcmp, uint64(ptr1+i), uint64(ptr2+i))
		if sign(int32(got)) != want {
			t.Errorf("strcmp(%d, %d) = %d, want %d",
				ptr1+i, ptr2+i, int32(got), want)
		}
	}
}

func Test_strncmp(t *testing.T) {
	const s1 = compareTest1
	const s2 = compareTest2

	ptr2 := len(memory) - len(s2) - 1

	clear(memory)
	copy(memory[ptr1:], s1)
	copy(memory[ptr2:], s2)

	for i := range len(s1) + 1 {
		for j := range len(s1) - i + 1 {
			want := strings.Compare(term(s1[i:i+j]), term(s2[i:i+j]))
			got := call(strncmp, uint64(ptr1+i), uint64(ptr2+i), uint64(j))
			if sign(int32(got)) != want {
				t.Errorf("strncmp(%d, %d, %d) = %d, want %d",
					ptr1+i, ptr2+i, j, int32(got), want)
			}
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

type searchTest struct {
	haystk string
	needle string
	out    int
}

var searchTests = []searchTest{
	{"", "", 0},
	{"", "a", -1},
	{"", "foo", -1},
	{"fo", "foo", -1},
	{"foo", "baz", -1},
	{"foo", "foo", 0},
	{"oofofoofooo", "f", 2},
	{"oofofoofooo", "foo", 4},
	{"barfoobarfoo", "foo", 3},
	{"foo", "", 0},
	{"foo", "o", 1},
	{"jrzm6jjhorimglljrea4w3rlgosts0w2gia17hno2td4qd1jz", "jz", 47},
	{"ekkuk5oft4eq0ocpacknhwouic1uua46unx12l37nioq9wbpnocqks6", "ks6", 52},
	{"999f2xmimunbuyew5vrkla9cpwhmxan8o98ec", "98ec", 33},
	{"9lpt9r98i04k8bz6c6dsrthb96bhi", "96bhi", 24},
	{"55u558eqfaod2r2gu42xxsu631xf0zobs5840vl", "5840vl", 33},
	{"", "a", -1},
	{"x", "a", -1},
	{"x", "x", 0},
	{"abc", "a", 0},
	{"abc", "b", 1},
	{"abc", "c", 2},
	{"abc", "x", -1},
	{"", "ab", -1},
	{"bc", "ab", -1},
	{"ab", "ab", 0},
	{"xab", "ab", 1},
	{"xab"[:2], "ab", -1},
	{"", "abc", -1},
	{"xbc", "abc", -1},
	{"abc", "abc", 0},
	{"xabc", "abc", 1},
	{"xabc"[:3], "abc", -1},
	{"xabxc", "abc", -1},
	{"", "abcd", -1},
	{"xbcd", "abcd", -1},
	{"abcd", "abcd", 0},
	{"xabcd", "abcd", 1},
	{"xyabcd"[:5], "abcd", -1},
	{"xbcqq", "abcqq", -1},
	{"abcqq", "abcqq", 0},
	{"xabcqq", "abcqq", 1},
	{"xyabcqq"[:6], "abcqq", -1},
	{"xabxcqq", "abcqq", -1},
	{"xabcqxq", "abcqq", -1},
	{"", "01234567", -1},
	{"32145678", "01234567", -1},
	{"01234567", "01234567", 0},
	{"x01234567", "01234567", 1},
	{"x0123456x01234567", "01234567", 9},
	{"xx01234567"[:9], "01234567", -1},
	{"", "0123456789", -1},
	{"3214567844", "0123456789", -1},
	{"0123456789", "0123456789", 0},
	{"x0123456789", "0123456789", 1},
	{"x012345678x0123456789", "0123456789", 11},
	{"xyz0123456789"[:12], "0123456789", -1},
	{"x01234567x89", "0123456789", -1},
	{"", "0123456789012345", -1},
	{"3214567889012345", "0123456789012345", -1},
	{"0123456789012345", "0123456789012345", 0},
	{"x0123456789012345", "0123456789012345", 1},
	{"x012345678901234x0123456789012345", "0123456789012345", 17},
	{"", "01234567890123456789", -1},
	{"32145678890123456789", "01234567890123456789", -1},
	{"01234567890123456789", "01234567890123456789", 0},
	{"x01234567890123456789", "01234567890123456789", 1},
	{"x0123456789012345678x01234567890123456789", "01234567890123456789", 21},
	{"xyz01234567890123456789"[:22], "01234567890123456789", -1},
	{"", "0123456789012345678901234567890", -1},
	{"321456788901234567890123456789012345678911", "0123456789012345678901234567890", -1},
	{"0123456789012345678901234567890", "0123456789012345678901234567890", 0},
	{"x0123456789012345678901234567890", "0123456789012345678901234567890", 1},
	{"x012345678901234567890123456789x0123456789012345678901234567890", "0123456789012345678901234567890", 32},
	{"xyz0123456789012345678901234567890"[:33], "0123456789012345678901234567890", -1},
	{"", "01234567890123456789012345678901", -1},
	{"32145678890123456789012345678901234567890211", "01234567890123456789012345678901", -1},
	{"01234567890123456789012345678901", "01234567890123456789012345678901", 0},
	{"x01234567890123456789012345678901", "01234567890123456789012345678901", 1},
	{"x0123456789012345678901234567890x01234567890123456789012345678901", "01234567890123456789012345678901", 33},
	{"xyz01234567890123456789012345678901"[:34], "01234567890123456789012345678901", -1},
	{"xxxxxx012345678901234567890123456789012345678901234567890123456789012", "012345678901234567890123456789012345678901234567890123456789012", 6},
	{"", "0123456789012345678901234567890123456789", -1},
	{"xx012345678901234567890123456789012345678901234567890123456789012", "0123456789012345678901234567890123456789", 2},
	{"xx012345678901234567890123456789012345678901234567890123456789012"[:41], "0123456789012345678901234567890123456789", -1},
	{"xx012345678901234567890123456789012345678901234567890123456789012", "0123456789012345678901234567890123456xxx", -1},
	{"xx0123456789012345678901234567890123456789012345678901234567890120123456789012345678901234567890123456xxx", "0123456789012345678901234567890123456xxx", 65},
	{"barfoobarfooyyyzzzyyyzzzyyyzzzyyyxxxzzzyyy", "x", 33},
	{"fofofofooofoboo", "oo", 7},
	{"fofofofofofoboo", "ob", 11},
	{"fofofofofofoboo", "boo", 12},
	{"fofofofofofoboo", "oboo", 11},
	{"fofofofofoooboo", "fooo", 8},
	{"fofofofofofoboo", "foboo", 10},
	{"fofofofofofoboo", "fofob", 8},
	{"fofofofofofofoffofoobarfoo", "foffof", 12},
	{"fofofofofoofofoffofoobarfoo", "foffof", 13},
	{"fofofofofofofoffofoobarfoo", "foffofo", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofo", 13},
	{"fofofofofoofofoffofoobarfoo", "foffofoo", 13},
	{"fofofofofofofoffofoobarfoo", "foffofoo", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofoob", 13},
	{"fofofofofofofoffofoobarfoo", "foffofoob", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofooba", 13},
	{"fofofofofofofoffofoobarfoo", "foffofooba", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofoobar", 13},
	{"fofofofofofofoffofoobarfoo", "foffofoobar", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofoobarf", 13},
	{"fofofofofofofoffofoobarfoo", "foffofoobarf", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofoobarfo", 13},
	{"fofofofofofofoffofoobarfoo", "foffofoobarfo", 12},
	{"fofofofofoofofoffofoobarfoo", "foffofoobarfoo", 13},
	{"fofofofofofofoffofoobarfoo", "foffofoobarfoo", 12},
	{"fofofofofoofofoffofoobarfoo", "ofoffofoobarfoo", 12},
	{"fofofofofofofoffofoobarfoo", "ofoffofoobarfoo", 11},
	{"fofofofofoofofoffofoobarfoo", "fofoffofoobarfoo", 11},
	{"fofofofofofofoffofoobarfoo", "fofoffofoobarfoo", 10},
	{"fofofofofoofofoffofoobarfoo", "foobars", -1},
	{"foofyfoobarfoobar", "y", 4},
	{"oooooooooooooooooooooo", "r", -1},
	{"oxoxoxoxoxoxoxoxoxoxoxoy", "oy", 22},
	{"oxoxoxoxoxoxoxoxoxoxoxox", "oy", -1},
	{"oxoxoxoxoxoxoxoxoxoxox☺", "☺", 22},
	{"xx0123456789012345678901234567890123456789012345678901234567890120123456789012345678901234567890123456xxx\xed\x9f\xc0", "\xed\x9f\xc0", 105},
	{"000000000000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000001", 5},
}

func Test_memmem(t *testing.T) {
	tt := append(searchTests,
		searchTest{"abcABCabc", "A", 3},
		searchTest{"fofofofofofo\x00foffofoobar", "foffof", 13},
		searchTest{"0000000000000000\x000123456789012345678901234567890", "0123456789012345", 17},
	)

	for i := range tt {
		ptr1 := uint64(len(memory) - len(tt[i].haystk))

		clear(memory)
		copy(memory[ptr1:], tt[i].haystk)
		copy(memory[ptr2:], tt[i].needle)

		var want uint64
		if tt[i].out >= 0 {
			want = ptr1 + uint64(tt[i].out)
		}

		got := call(memmem,
			uint64(ptr1), uint64(len(tt[i].haystk)),
			uint64(ptr2), uint64(len(tt[i].needle)))
		if got != want {
			t.Errorf("memmem(%q, %q) = %d, want %d",
				tt[i].haystk, tt[i].needle,
				uint32(got), uint32(want))
		}
	}
}

func Test_strstr(t *testing.T) {
	tt := append(searchTests,
		searchTest{"abcABCabc", "A", 3},
		searchTest{"fofofofofofo\x00foffofoobar", "foffof", -1},
		searchTest{"0000000000000000\x000123456789012345678901234567890", "0123456789012345", -1},
	)

	for i := range tt {
		ptr1 := uint64(len(memory) - len(tt[i].haystk) - 1)

		clear(memory)
		copy(memory[ptr1:], tt[i].haystk)
		copy(memory[ptr2:], tt[i].needle)

		var want uint64
		if tt[i].out >= 0 {
			want = ptr1 + uint64(tt[i].out)
		}

		got := call(strstr, uint64(ptr1), uint64(ptr2))
		if got != want {
			t.Errorf("strstr(%q, %q) = %d, want %d",
				tt[i].haystk, tt[i].needle,
				uint32(got), uint32(want))
		}
	}
}

func Test_strcasestr(t *testing.T) {
	tt := append(searchTests[1:],
		searchTest{"A", "a", 0},
		searchTest{"a", "A", 0},
		searchTest{"Z", "z", 0},
		searchTest{"z", "Z", 0},
		searchTest{"@", "`", -1},
		searchTest{"`", "@", -1},
		searchTest{"[", "{", -1},
		searchTest{"{", "[", -1},
		searchTest{"abcABCabc", "A", 0},
		searchTest{"fofofofofofofoffofoobarfoo", "FoFFoF", 12},
		searchTest{"fofofofofofofOffOfoobarfoo", "FoFFoF", 12},
		searchTest{"fofofofofofo\x00foffofoobar", "foffof", -1},
		searchTest{"0000000000000000\x000123456789012345678901234567890", "0123456789012345", -1},
	)

	for i := range tt {
		ptr1 := uint64(len(memory) - len(tt[i].haystk) - 1)

		clear(memory)
		copy(memory[ptr1:], tt[i].haystk)
		copy(memory[ptr2:], tt[i].needle)

		var want uint64
		if tt[i].out >= 0 {
			want = ptr1 + uint64(tt[i].out)
		}

		got := call(strcasestr, uint64(ptr1), uint64(ptr2))
		if got != want {
			t.Errorf("strcasestr(%q, %q) = %d, want %d",
				tt[i].haystk, tt[i].needle,
				uint32(got), uint32(want))
		}
	}
}

func Fuzz_memchr(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string, c, i byte) {
		if len(s) > 128 || int(i) > len(s) {
			t.SkipNow()
		}
		copy(memory[ptr1:], s)

		got := call(memchr, ptr1+uint64(i), uint64(c), uint64(len(s)-int(i)))
		want := strings.IndexByte(s[i:], c)
		if want >= 0 {
			want = ptr1 + int(i) + want
		} else {
			want = 0
		}

		if uint32(got) != uint32(want) {
			t.Errorf("memchr(%q, %q) = %d, want %d",
				s[i:], c, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strchr(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string, c, i byte) {
		if len(s) > 128 || int(i) > len(s) {
			t.SkipNow()
		}
		copy(memory[ptr1:], s)
		memory[ptr1+len(s)] = 0

		got := call(strchr, ptr1+uint64(i), uint64(c))
		want := bytes.IndexByte(term1(memory[ptr1+uint64(i):]), c)
		if want >= 0 {
			want = ptr1 + int(i) + want
		} else {
			want = 0
		}

		if uint32(got) != uint32(want) {
			t.Errorf("strchr(%q, %q) = %d, want %d",
				s[i:], c, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strrchr(f *testing.F) {
	f.Fuzz(func(t *testing.T, s string, c, i byte) {
		if len(s) > 128 || int(i) > len(s) {
			t.SkipNow()
		}
		copy(memory[ptr1:], s)
		memory[ptr1+len(s)] = 0

		got := call(strrchr, ptr1+uint64(i), uint64(c))
		want := bytes.LastIndexByte(term1(memory[ptr1+uint64(i):]), c)
		if want >= 0 {
			want = ptr1 + int(i) + want
		} else {
			want = 0
		}

		if uint32(got) != uint32(want) {
			t.Errorf("strrchr(%q, %q) = %d, want %d",
				s[i:], c, uint32(got), uint32(want))
		}
	})
}

func Fuzz_memcmp(f *testing.F) {
	const s1 = compareTest1
	const s2 = compareTest2

	for i := range len(compareTest1) + 1 {
		f.Add(s1[i:], s2[i:])
	}

	f.Fuzz(func(t *testing.T, s1, s2 string) {
		if len(s1) > 128 || len(s1) != len(s2) {
			t.SkipNow()
		}
		copy(memory[ptr1:], s1)
		copy(memory[ptr2:], s2)

		got := call(memcmp, uint64(ptr1), uint64(ptr2), uint64(len(s1)))
		want := strings.Compare(s1, s2)

		if sign(int32(got)) != want {
			t.Errorf("memcmp(%q, %q) = %d, want %d",
				s1, s2, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strcmp(f *testing.F) {
	const s1 = compareTest1
	const s2 = compareTest2

	for i := range len(compareTest1) + 1 {
		f.Add(term(s1[i:]), term(s2[i:]))
	}

	f.Fuzz(func(t *testing.T, s1, s2 string) {
		if len(s1) > 128 || len(s2) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], s1)
		copy(memory[ptr2:], s2)
		memory[ptr1+len(s1)] = 0
		memory[ptr2+len(s2)] = 0

		got := call(strcmp, uint64(ptr1), uint64(ptr2))
		want := strings.Compare(term(s1), term(s2))

		if sign(int32(got)) != want {
			t.Errorf("strcmp(%q, %q) = %d, want %d",
				s1, s2, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strncmp(f *testing.F) {
	const s1 = compareTest1
	const s2 = compareTest2

	for i := range len(compareTest1) + 1 {
		f.Add(term(s1[i:]), term(s2[i:]), byte(len(s1)))
	}

	f.Fuzz(func(t *testing.T, s1, s2 string, n byte) {
		if len(s1) > 128 || len(s2) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], s1)
		copy(memory[ptr2:], s2)
		memory[ptr1+len(s1)] = 0
		memory[ptr2+len(s2)] = 0

		got := call(strncmp, uint64(ptr1), uint64(ptr2), uint64(n))
		want := bytes.Compare(
			term(memory[ptr1:][:n]),
			term(memory[ptr2:][:n]))

		if sign(int32(got)) != want {
			t.Errorf("strncmp(%q, %q, %d) = %d, want %d",
				s1, s2, n, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strcasecmp(f *testing.F) {
	const s1 = compareTest1
	const s2 = compareTest2

	for i := range len(compareTest1) + 1 {
		f.Add(term(s1[i:]), term(s2[i:]))
	}

	f.Fuzz(func(t *testing.T, s1, s2 string) {
		if len(s1) > 128 || len(s2) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], s1)
		copy(memory[ptr2:], s2)
		memory[ptr1+len(s1)] = 0
		memory[ptr2+len(s2)] = 0

		got := call(strcasecmp, uint64(ptr1), uint64(ptr2))
		want := bytes.Compare(
			lower(term(memory[ptr1:])),
			lower(term(memory[ptr2:])))

		if sign(int32(got)) != want {
			t.Errorf("strcasecmp(%q, %q) = %d, want %d",
				s1, s2, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strncasecmp(f *testing.F) {
	const s1 = compareTest1
	const s2 = compareTest2

	for i := range len(compareTest1) + 1 {
		f.Add(term(s1[i:]), term(s2[i:]), byte(len(s1)))
	}

	f.Fuzz(func(t *testing.T, s1, s2 string, n byte) {
		if len(s1) > 128 || len(s2) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], s1)
		copy(memory[ptr2:], s2)
		memory[ptr1+len(s1)] = 0
		memory[ptr2+len(s2)] = 0

		got := call(strncasecmp, uint64(ptr1), uint64(ptr2), uint64(n))
		want := bytes.Compare(
			lower(term(memory[ptr1:][:n])),
			lower(term(memory[ptr2:][:n])))

		if sign(int32(got)) != want {
			t.Errorf("strncasecmp(%q, %q, %d) = %d, want %d",
				s1, s2, n, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strspn(f *testing.F) {
	for _, t := range searchTests {
		f.Add(t.haystk, t.needle)
	}

	f.Fuzz(func(t *testing.T, s, chars string) {
		if len(s) > 128 || len(chars) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], s)
		copy(memory[ptr2:], chars)
		memory[ptr1+len(s)] = 0
		memory[ptr2+len(chars)] = 0

		got := call(strspn, uint64(ptr1), uint64(ptr2))

		s = term(s)
		chars = term(chars)
		want := strings.IndexFunc(s, func(r rune) bool {
			if uint32(r) >= utf8.RuneSelf {
				t.Skip()
			}
			return strings.IndexByte(chars, byte(r)) < 0
		})
		if want < 0 {
			want = len(s)
		}

		if uint32(got) != uint32(want) {
			t.Errorf("strspn(%q, %q) = %d, want %d",
				s, chars, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strcspn(f *testing.F) {
	for _, t := range searchTests {
		f.Add(t.haystk, t.needle)
	}

	f.Fuzz(func(t *testing.T, s, chars string) {
		if len(s) > 128 || len(chars) > 128 {
			t.SkipNow()
		}
		if strings.ContainsFunc(chars, func(r rune) bool {
			return uint32(r) >= utf8.RuneSelf
		}) {
			t.SkipNow()
		}
		copy(memory[ptr1:], s)
		copy(memory[ptr2:], chars)
		memory[ptr1+len(s)] = 0
		memory[ptr2+len(chars)] = 0

		got := call(strcspn, uint64(ptr1), uint64(ptr2))

		s = term(s)
		chars = term(chars)
		want := strings.IndexAny(s, chars)
		if want < 0 {
			want = len(s)
		}

		if uint32(got) != uint32(want) {
			t.Errorf("strcspn(%q, %q) = %d, want %d",
				s, chars, uint32(got), uint32(want))
		}
	})
}

func Fuzz_memmem(f *testing.F) {
	tt := append(searchTests,
		searchTest{"abcABCabc", "A", 3},
		searchTest{"fofofofofofo\x00foffofoobar", "foffof", 13},
		searchTest{"0000000000000000\x000123456789012345678901234567890", "0123456789012345", 17},
	)

	for _, t := range tt {
		f.Add(t.haystk, t.needle)
	}

	f.Fuzz(func(t *testing.T, haystk, needle string) {
		if len(haystk) > 128 || len(needle) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], haystk)
		copy(memory[ptr2:], needle)

		got := call(memmem,
			uint64(ptr1), uint64(len(haystk)),
			uint64(ptr2), uint64(len(needle)))

		want := strings.Index(haystk, needle)
		if want >= 0 {
			want = ptr1 + want
		} else {
			want = 0
		}

		if uint32(got) != uint32(want) {
			t.Errorf("memmem(%q, %q) = %d, want %d",
				haystk, needle, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strstr(f *testing.F) {
	tt := append(searchTests,
		searchTest{"abcABCabc", "A", 3},
		searchTest{"fofofofofofo\x00foffofoobar", "foffof", -1},
		searchTest{"0000000000000000\x000123456789012345678901234567890", "0123456789012345", -1},
	)

	for _, t := range tt {
		f.Add(t.haystk, t.needle)
	}

	f.Fuzz(func(t *testing.T, haystk, needle string) {
		if len(haystk) > 128 || len(needle) > 128 {
			t.SkipNow()
		}
		copy(memory[ptr1:], haystk)
		copy(memory[ptr2:], needle)
		memory[ptr1+len(haystk)] = 0
		memory[ptr2+len(needle)] = 0

		got := call(strstr, uint64(ptr1), uint64(ptr2))

		want := strings.Index(term(haystk), term(needle))
		if want >= 0 {
			want = ptr1 + want
		} else {
			want = 0
		}

		if uint32(got) != uint32(want) {
			t.Errorf("strstr(%q, %q) = %d, want %d",
				haystk, needle, uint32(got), uint32(want))
		}
	})
}

func Fuzz_strcasestr(f *testing.F) {
	tt := append(searchTests,
		searchTest{"A", "a", 0},
		searchTest{"a", "A", 0},
		searchTest{"Z", "z", 0},
		searchTest{"z", "Z", 0},
		searchTest{"@", "`", -1},
		searchTest{"`", "@", -1},
		searchTest{"[", "{", -1},
		searchTest{"{", "[", -1},
		searchTest{"abcABCabc", "A", 0},
		searchTest{"fofofofofofofoffofoobarfoo", "FoFFoF", 12},
		searchTest{"fofofofofofofOffOfoobarfoo", "FoFFoF", 12},
		searchTest{"fofofofofofo\x00foffofoobar", "foffof", -1},
		searchTest{"0000000000000000\x000123456789012345678901234567890", "0123456789012345", -1},
	)

	for _, t := range tt {
		f.Add(t.haystk, t.needle)
	}

	f.Fuzz(func(t *testing.T, haystk, needle string) {
		if len(haystk) > 128 || len(needle) > 128 {
			t.SkipNow()
		}
		if len(needle) == 0 {
			t.Skip("musl bug")
		}
		copy(memory[ptr1:], haystk)
		copy(memory[ptr2:], needle)
		memory[ptr1+len(haystk)] = 0
		memory[ptr2+len(needle)] = 0

		got := call(strcasestr, uint64(ptr1), uint64(ptr2))

		want := bytes.Index(
			lower(term(memory[ptr1:])),
			lower(term(memory[ptr2:])))
		if want >= 0 {
			want = ptr1 + want
		} else {
			want = 0
		}

		if uint32(got) != uint32(want) {
			t.Errorf("strcasestr(%q, %q) = %d, want %d",
				haystk, needle, uint32(got), uint32(want))
		}
	})
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

func fill(s []byte, v byte) {
	for i := range s {
		s[i] = v
	}
}

func lower(s []byte) []byte {
	for i, c := range s {
		if 'A' <= c && c <= 'Z' {
			s[i] = c - 'A' + 'a'
		}
	}
	return s
}

func term[T interface{ []byte | string }](s T) T {
	for i, c := range []byte(s) {
		if c == 0 {
			return s[:i]
		}
	}
	return s
}

func term1[T interface{ []byte | string }](s T) T {
	for i, c := range []byte(s) {
		if c == 0 {
			return s[:i+1]
		}
	}
	return s
}
