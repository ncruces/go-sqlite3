package mptest

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	_ "unsafe"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	_ "github.com/ncruces/go-sqlite3"
)

//go:embed testdata/mptest.wasm
var binary []byte

//go:embed testdata/*.*test
var scripts embed.FS

//go:linkname vfsNewEnvModuleBuilder github.com/ncruces/go-sqlite3.vfsNewEnvModuleBuilder
func vfsNewEnvModuleBuilder(r wazero.Runtime) wazero.HostModuleBuilder

var (
	rt        wazero.Runtime
	module    wazero.CompiledModule
	config    wazero.ModuleConfig
	instances atomic.Uint64
	log       *logger
)

func init() {
	ctx := context.TODO()

	rt = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	env := vfsNewEnvModuleBuilder(rt)
	env.NewFunctionBuilder().WithFunc(system).Export("system")
	_, err := env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}

	module, err = rt.CompileModule(ctx, binary)
	if err != nil {
		panic(err)
	}

	fs, err := fs.Sub(scripts, "testdata")
	if err != nil {
		panic(err)
	}

	config = wazero.NewModuleConfig().WithFS(fs).
		WithSysWalltime().WithSysNanotime().WithSysNanosleep().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader)
}

func system(ctx context.Context, mod api.Module, ptr uint32) uint32 {
	buf, _ := mod.Memory().Read(ptr, mod.Memory().Size()-ptr)
	buf = buf[:bytes.IndexByte(buf, 0)]

	args := strings.Split(string(buf), " ")
	for i := range args {
		a, err := strconv.Unquote(args[i])
		if err == nil {
			args[i] = a
		}
	}
	args = args[:len(args)-1]

	cfg := config.WithArgs(args...).
		WithStdout(log).WithStderr(log).WithName(instanceName())
	go rt.InstantiateModule(ctx, module, cfg)
	return 0
}

func Test_config01(t *testing.T) {
	log = &logger{T: t}
	ctx := context.TODO()
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config.WithArgs("mptest", name, "config01.test").
		WithStdout(log).WithStderr(log).WithName(instanceName())
	_, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
}

func Test_config02(t *testing.T) {
	t.Skip() // TODO: remove
	log = &logger{T: t}
	ctx := context.TODO()
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config.WithArgs("mptest", name, "config02.test").
		WithStdout(log).WithStderr(log).WithName(instanceName())
	_, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
}

func Test_crash01(t *testing.T) {
	t.Skip() // TODO: remove
	log = &logger{T: t}
	ctx := context.TODO()
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config.WithArgs("mptest", name, "crash01.test").
		WithStdout(log).WithStderr(log).WithName(instanceName())
	_, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
}

func Test_multiwrite01(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // TODO: remove
	}
	log = &logger{T: t}
	ctx := context.TODO()
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config.WithArgs("mptest", name, "multiwrite01.test").
		WithStdout(log).WithStderr(log).WithName(instanceName())
	_, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
}

type logger struct {
	*testing.T
	buf []byte
	mtx sync.Mutex
}

func (l *logger) Write(p []byte) (n int, err error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	l.buf = append(l.buf, p...)
	for {
		before, after, found := bytes.Cut(l.buf, []byte("\n"))
		if !found {
			return len(p), nil
		}
		l.Logf("%s", before)
		l.buf = after
	}
}

func instanceName() string {
	return strconv.FormatUint(instances.Add(1), 10)
}
