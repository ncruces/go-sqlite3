package speedtest1

import (
	"bufio"
	"context"
	_ "embed"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	_ "unsafe"

	_ "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/internal/testfs"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
	_ "github.com/ncruces/go-sqlite3/vfs/mvcc"
	_ "github.com/ncruces/go-sqlite3/vfs/xts"
)

const ptrlen = sqlite3_wrap.PtrLen

type ptr_t = sqlite3_wrap.Ptr_t

//go:linkname createWrapper github.com/ncruces/go-sqlite3.createWrapper
func createWrapper(ctx context.Context) (*sqlite3_wrap.Wrapper, error)

var options []string

func TestMain(m *testing.M) {
	testfs.Stdout = bufio.NewWriter(os.Stdout)
	testfs.Stderr = bufio.NewWriter(os.Stderr)
	initFlags()
	os.Exit(m.Run())
}

func initFlags() {
	i := 1
	options = append(options, "speedtest1")
	for _, arg := range os.Args[1:] {
		switch {
		case strings.HasPrefix(arg, "-test."):
			// keep test flags
			os.Args[i] = arg
			i++
		case arg == "--":
			// ignore this
		default:
			// collect everything else
			options = append(options, arg)
		}
	}
	os.Args = os.Args[:i]
	flag.Parse()
}

func runBenchmark(b *testing.B, args ...string) {
	if b.N == 1 {
		return
	}

	wrp, err := createWrapper(b.Context())
	if err != nil {
		b.Fatal(err)
	}
	defer wrp.Close()

	args = append(options, args...)

	argv := wrp.New(int64(ptrlen * len(args)))
	for i, a := range args {
		wrp.Write32(argv+ptr_t(i)*ptrlen, uint32(wrp.NewString(a)))
	}

	wrp.Xmain_speedtest1(int32(len(args)), int32(argv))
}

func Benchmark_speedtest1(b *testing.B) {
	name := filepath.Join(b.TempDir(), "test.db")
	runBenchmark(b, "--size", strconv.Itoa(b.N), name)
}

func Benchmark_adiantum(b *testing.B) {
	name := "file:" + filepath.Join(b.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	runBenchmark(b, "--vfs", "adiantum", "--size", strconv.Itoa(b.N), name)
}

func Benchmark_xts(b *testing.B) {
	name := "file:" + filepath.Join(b.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	runBenchmark(b, "--vfs", "xts", "--size", strconv.Itoa(b.N), name)
}
