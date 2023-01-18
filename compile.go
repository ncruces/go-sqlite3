package sqlite3

import (
	"context"
	"os"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
)

// Configure SQLite.
var (
	Binary []byte // Binary to load.
	Path   string // Path to load the binary from.
)

var (
	once    sync.Once
	wasm    wazero.Runtime
	module  wazero.CompiledModule
	counter atomic.Uint64
)

func compile() {
	ctx := context.Background()

	wasm = wazero.NewRuntime(ctx)

	if err := vfsInstantiate(ctx, wasm); err != nil {
		panic(err)
	}

	if Binary == nil && Path != "" {
		if bin, err := os.ReadFile(Path); err != nil {
			panic(err)
		} else {
			Binary = bin
		}
	}

	if m, err := wasm.CompileModule(ctx, Binary); err != nil {
		panic(err)
	} else {
		module = m
	}
}
