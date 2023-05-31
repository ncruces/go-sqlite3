package mptest

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ncruces/go-sqlite3/sqlite3vfs"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed testdata/mptest.wasm
var binary []byte

//go:embed testdata/*.*test
var scripts embed.FS

var (
	rt        wazero.Runtime
	module    wazero.CompiledModule
	instances atomic.Uint64
	memory    = sqlite3vfs.MemoryVFS{}
)

func init() {
	ctx := context.TODO()

	rt = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	env := sqlite3vfs.ExportHostFunctions(rt.NewHostModuleBuilder("env"))
	env.NewFunctionBuilder().WithFunc(system).Export("system")
	_, err := env.Instantiate(ctx)
	if err != nil {
		panic(err)
	}

	module, err = rt.CompileModule(ctx, binary)
	if err != nil {
		panic(err)
	}

	sqlite3vfs.Register("memvfs", memory)
}

func config(ctx context.Context) wazero.ModuleConfig {
	name := strconv.FormatUint(instances.Add(1), 10)
	log := ctx.Value(logger{}).(io.Writer)
	fs, err := fs.Sub(scripts, "testdata")
	if err != nil {
		panic(err)
	}

	return wazero.NewModuleConfig().
		WithName(name).WithStdout(log).WithStderr(log).WithFS(fs).
		WithSysWalltime().WithSysNanotime().WithSysNanosleep().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader)
}

func system(ctx context.Context, mod api.Module, ptr uint32) uint32 {
	buf, _ := mod.Memory().Read(ptr, mod.Memory().Size()-ptr)
	buf = buf[:bytes.IndexByte(buf, 0)]

	var memvfs, journal, timeout bool
	args := strings.Split(string(buf), " ")
	for i := range args {
		args[i] = strings.Trim(args[i], `"`)
		switch args[i] {
		case "memvfs":
			memvfs = true
		case "--timeout":
			timeout = true
		case "--journalmode":
			journal = true
		}
	}
	args = args[:len(args)-1]
	if memvfs {
		if !timeout {
			args = append(args, "--timeout", "1000")
		}
		if !journal {
			args = append(args, "--journalmode", "memory")
		}
	}

	cfg := config(ctx).WithArgs(args...)
	go func() {
		ctx, vfs := sqlite3vfs.NewContext(ctx)
		mod, _ := rt.InstantiateModule(ctx, module, cfg)
		mod.Close(ctx)
		vfs.Close()
	}()
	return 0
}

func Test_config01(t *testing.T) {
	ctx, vfs := sqlite3vfs.NewContext(newContext(t))
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config(ctx).WithArgs("mptest", name, "config01.test")
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
	mod.Close(ctx)
	vfs.Close()
}

func Test_config02(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI")
	}

	ctx, vfs := sqlite3vfs.NewContext(newContext(t))
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config(ctx).WithArgs("mptest", name, "config02.test")
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
	mod.Close(ctx)
	vfs.Close()
}

func Test_crash01(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	ctx, vfs := sqlite3vfs.NewContext(newContext(t))
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config(ctx).WithArgs("mptest", name, "crash01.test")
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
	mod.Close(ctx)
	vfs.Close()
}

func Test_multiwrite01(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	ctx, vfs := sqlite3vfs.NewContext(newContext(t))
	name := filepath.Join(t.TempDir(), "test.db")
	cfg := config(ctx).WithArgs("mptest", name, "multiwrite01.test")
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
	mod.Close(ctx)
	vfs.Close()
}

func Test_config01_memory(t *testing.T) {
	memory["test.db"] = new(sqlite3vfs.MemoryDB)
	ctx, vfs := sqlite3vfs.NewContext(newContext(t))
	cfg := config(ctx).WithArgs("mptest", "test.db",
		"config01.test",
		"--vfs", "memvfs",
		"--timeout", "1000",
		"--journalmode", "memory")
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
	mod.Close(ctx)
	vfs.Close()
}

func Test_multiwrite01_memory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	memory["test.db"] = new(sqlite3vfs.MemoryDB)
	ctx, vfs := sqlite3vfs.NewContext(newContext(t))
	cfg := config(ctx).WithArgs("mptest", "test.db",
		"multiwrite01.test",
		"--vfs", "memvfs",
		"--timeout", "1000",
		"--journalmode", "memory")
	mod, err := rt.InstantiateModule(ctx, module, cfg)
	if err != nil {
		t.Error(err)
	}
	mod.Close(ctx)
	vfs.Close()
}

func newContext(t *testing.T) context.Context {
	return context.WithValue(context.Background(), logger{}, &testWriter{T: t})
}

type logger struct{}

type testWriter struct {
	// +checklocks:mtx
	*testing.T
	// +checklocks:mtx
	buf []byte
	mtx sync.Mutex
}

func (l *testWriter) Write(p []byte) (n int, err error) {
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
