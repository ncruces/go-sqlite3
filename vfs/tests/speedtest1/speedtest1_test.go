package speedtest1

import (
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/rand"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	_ "embed"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

//go:embed testdata/speedtest1.wasm.bz2
var compressed string

var (
	rt      wazero.Runtime
	module  wazero.CompiledModule
	output  bytes.Buffer
	options []string
)

func TestMain(m *testing.M) {
	initFlags()

	ctx := context.Background()
	rt = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	env := vfs.ExportHostFunctions(rt.NewHostModuleBuilder("env"))
	_, err := env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}

	if !strings.HasPrefix(compressed, "BZh") {
		panic("Please use Git LFS to clone this repo: https://git-lfs.com/")
	}
	binary, err := io.ReadAll(bzip2.NewReader(strings.NewReader(compressed)))
	if err != nil {
		panic(err)
	}

	module, err = rt.CompileModule(ctx, binary)
	if err != nil {
		panic(err)
	}

	code := m.Run()
	defer os.Exit(code)
	io.Copy(os.Stderr, &output)
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
		default:
			// collect everything else
			options = append(options, arg)
		}
	}
	os.Args = os.Args[:i]
	flag.Parse()
}

func Benchmark_speedtest1(b *testing.B) {
	output.Reset()
	ctx := util.NewContext(context.Background())
	name := filepath.Join(b.TempDir(), "test.db")
	args := append(options, "--size", strconv.Itoa(b.N), name)
	cfg := wazero.NewModuleConfig().
		WithArgs(args...).WithName("speedtest1").
		WithStdout(&output).WithStderr(&output).
		WithSysWalltime().WithSysNanotime().WithSysNanosleep().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader)
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		b.Fatal(err)
	}
	mod.Close(ctx)
}
