package speedtest1

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
	_ "github.com/ncruces/go-sqlite3/vfs/xts"
)

var (
	rt      wazero.Runtime
	module  wazero.CompiledModule
	output  bytes.Buffer
	options []string
)

func TestMain(m *testing.M) {
	initFlags()

	ctx := context.Background()
	cfg := wazero.NewRuntimeConfig().WithMemoryLimitPages(2048)
	rt = wazero.NewRuntimeWithConfig(ctx, cfg)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	env := vfs.ExportHostFunctions(rt.NewHostModuleBuilder("env"))
	_, err := env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}

	binary, err := os.ReadFile("wasm/speedtest1.wasm")
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

func Benchmark_adiantum(b *testing.B) {
	output.Reset()
	ctx := util.NewContext(context.Background())
	name := "file:" + filepath.Join(b.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	args := append(options, "--vfs", "adiantum", "--size", strconv.Itoa(b.N), name)
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

func Benchmark_xts(b *testing.B) {
	output.Reset()
	ctx := util.NewContext(context.Background())
	name := "file:" + filepath.Join(b.TempDir(), "test.db") +
		"?hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	args := append(options, "--vfs", "xts", "--size", strconv.Itoa(b.N), name)
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
