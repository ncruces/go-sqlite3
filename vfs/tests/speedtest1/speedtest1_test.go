package speedtest1

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	_ "embed"

	"github.com/stealthrocket/wzprof"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/ncruces/go-sqlite3/vfs"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

//go:embed testdata/speedtest1.wasm
var binary []byte

var (
	rt      wazero.Runtime
	module  wazero.CompiledModule
	output  bytes.Buffer
	options []string
	cpuprof string
	memprof string
)

func TestMain(m *testing.M) {
	initFlags()

	ctx := context.Background()
	ctx, prof, cpu, mem := setupProfiling(ctx)

	rt = wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	env := vfs.ExportHostFunctions(rt.NewHostModuleBuilder("env"))
	_, err := env.Instantiate(ctx)
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
	saveProfiles(module, prof, cpu, mem)
}

func initFlags() {
	i := 1
	wzprof := false
	options = append(options, "speedtest1")
	for _, arg := range os.Args[1:] {
		switch {
		case strings.HasPrefix(arg, "-test."):
			// keep test flags
			os.Args[i] = arg
			i++
		case strings.HasSuffix(arg, "-wzprof"):
			// collect guest profile
			wzprof = true
		default:
			// collect everything else
			options = append(options, arg)
		}
	}
	os.Args = os.Args[:i]
	flag.Parse()

	if wzprof {
		var f *flag.Flag
		f = flag.Lookup("test.cpuprofile")
		cpuprof = f.Value.String()
		f.Value.Set("")
		f = flag.Lookup("test.memprofile")
		memprof = f.Value.String()
		f.Value.Set("")
	}
}

func setupProfiling(ctx context.Context) (context.Context, *wzprof.Profiling, *wzprof.CPUProfiler, *wzprof.MemoryProfiler) {
	if cpuprof == "" && memprof == "" {
		return ctx, nil, nil, nil
	}

	var cpu *wzprof.CPUProfiler
	var mem *wzprof.MemoryProfiler
	prof := wzprof.ProfilingFor(binary)

	var listeners []experimental.FunctionListenerFactory
	if cpuprof != "" {
		cpu = prof.CPUProfiler()
		listeners = append(listeners, cpu)
		cpu.StartProfile()
	}
	if memprof != "" {
		mem = prof.MemoryProfiler()
		listeners = append(listeners, mem)
	}
	if listeners != nil {
		ctx = context.WithValue(ctx,
			experimental.FunctionListenerFactoryKey{},
			experimental.MultiFunctionListenerFactory(listeners...))
	}
	return ctx, prof, cpu, mem
}

func saveProfiles(module wazero.CompiledModule, prof *wzprof.Profiling, cpu *wzprof.CPUProfiler, mem *wzprof.MemoryProfiler) {
	if cpu == nil && mem == nil {
		return
	}

	log.SetOutput(io.Discard)
	err := prof.Prepare(module)
	if err != nil {
		panic(err)
	}

	if cpu != nil {
		prof := cpu.StopProfile(1)
		err := wzprof.WriteProfile(cpuprof, prof)
		if err != nil {
			panic(err)
		}
	}

	if mem != nil {
		prof := mem.NewProfile(1)
		err := wzprof.WriteProfile(memprof, prof)
		if err != nil {
			panic(err)
		}
	}
}

func Benchmark_speedtest1(b *testing.B) {
	output.Reset()
	ctx, vfs := vfs.NewContext(context.Background())
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
	mod.Close(ctx)
	vfs.Close()
}
