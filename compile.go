package sqlite3

import (
	"context"
	"crypto/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Configure SQLite WASM.
//
// Importing package embed initializes these
// with an appropriate build of SQLite:
//
//	import _ "github.com/ncruces/go-sqlite3/embed"
var (
	Binary []byte // WASM binary to load.
	Path   string // Path to load the binary from.
)

var sqlite3 sqlite3Runtime

type sqlite3Runtime struct {
	once      sync.Once
	runtime   wazero.Runtime
	compiled  wazero.CompiledModule
	instances atomic.Uint64
	err       error
}

func instantiateModule() (m *module, err error) {
	ctx := context.Background()

	sqlite3.once.Do(func() { sqlite3.compileModule(ctx) })
	if sqlite3.err != nil {
		return nil, sqlite3.err
	}

	name := "sqlite3-" + strconv.FormatUint(sqlite3.instances.Add(1), 10)

	cfg := wazero.NewModuleConfig().WithName(name).
		WithSysWalltime().WithSysNanotime().WithSysNanosleep().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader)

	mod, err := sqlite3.runtime.InstantiateModule(ctx, sqlite3.compiled, cfg)
	if err != nil {
		return nil, err
	}

	module := &module{
		Module: mod,
		ctx:    ctx,
		mem:    memory{mod},
	}

	err = module.loadAPI()
	if err != nil {
		return nil, err
	}
	return module, nil
}

func (s *sqlite3Runtime) compileModule(ctx context.Context) {
	s.runtime = wazero.NewRuntime(ctx)
	vfsInstantiate(ctx, s.runtime)

	bin := Binary
	if bin == nil && Path != "" {
		bin, s.err = os.ReadFile(Path)
		if s.err != nil {
			return
		}
	}
	if bin == nil {
		s.err = binaryErr
		return
	}

	s.compiled, s.err = s.runtime.CompileModule(ctx, bin)
}

type module struct {
	api.Module

	ctx context.Context
	mem memory
	api sqliteAPI
}
