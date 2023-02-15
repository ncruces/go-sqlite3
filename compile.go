package sqlite3

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Configure SQLite.
var (
	Binary []byte // Binary to load.
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

func (s *sqlite3Runtime) instantiateModule(ctx context.Context) (api.Module, error) {
	s.once.Do(func() { s.compileModule(ctx) })
	if s.err != nil {
		return nil, s.err
	}

	cfg := wazero.NewModuleConfig().
		WithName("sqlite3-" + strconv.FormatUint(s.instances.Add(1), 10))
	return s.runtime.InstantiateModule(ctx, s.compiled, cfg)
}

func (s *sqlite3Runtime) compileModule(ctx context.Context) {
	s.runtime = wazero.NewRuntime(ctx)
	s.err = vfsInstantiate(ctx, s.runtime)
	if s.err != nil {
		return
	}

	bin := Binary
	if bin == nil && Path != "" {
		bin, s.err = os.ReadFile(Path)
		if s.err != nil {
			return
		}
	}

	s.compiled, s.err = s.runtime.CompileModule(ctx, bin)
}
