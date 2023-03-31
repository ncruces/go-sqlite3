package speedtest1

import (
	"bytes"
	"context"
	"crypto/rand"
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

	"github.com/ncruces/go-sqlite3/internal/vfs"
)

//go:embed testdata/speedtest1.wasm
var binary []byte

var (
	rt      wazero.Runtime
	module  wazero.CompiledModule
	output  bytes.Buffer
	options []string
)

func init() {
	ctx := context.TODO()

	rt = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	env := vfs.NewEnvModuleBuilder(rt)
	_, err := env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}

	module, err = rt.CompileModule(ctx, binary)
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	i := 1
	options = append(options, "speedtest1")
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-test.") {
			os.Args[i] = arg
			i++
		} else {
			options = append(options, arg)
		}
	}
	os.Args = os.Args[:i]

	code := m.Run()
	io.Copy(os.Stderr, &output)
	os.Exit(code)
}

func Benchmark_speedtest1(b *testing.B) {
	output.Reset()
	ctx, vfs := vfs.Context(context.Background())
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
		b.Error(err)
	}
	vfs.Close()
	mod.Close(ctx)
}
