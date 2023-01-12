package sqlite3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
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
	wasi_snapshot_preview1.MustInstantiate(ctx, wasm)

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

type Conn struct {
	handle uint32
	module api.Module
	memory api.Memory
	api    sqliteAPI
}

func Open(name string, flags uint64, vfs string) (*Conn, error) {
	once.Do(compile)

	ctx := context.TODO()

	cfg := wazero.NewModuleConfig().
		WithName("sqlite3-" + strconv.FormatUint(counter.Add(1), 10))
	module, err := wasm.InstantiateModule(ctx, module, cfg)
	if err != nil {
		return nil, err
	}

	c := Conn{
		module: module,
		memory: module.Memory(),
		api: sqliteAPI{
			malloc:      module.ExportedFunction("malloc"),
			free:        module.ExportedFunction("free"),
			errmsg:      module.ExportedFunction("sqlite3_errmsg"),
			open:        module.ExportedFunction("sqlite3_open_v2"),
			close:       module.ExportedFunction("sqlite3_close"),
			prepare:     module.ExportedFunction("sqlite3_prepare_v2"),
			exec:        module.ExportedFunction("sqlite3_exec"),
			step:        module.ExportedFunction("sqlite3_step"),
			columnText:  module.ExportedFunction("sqlite3_column_text"),
			columnInt:   module.ExportedFunction("sqlite3_column_int64"),
			columnFloat: module.ExportedFunction("sqlite3_column_double"),
		},
	}

	namePtr := c.newString(name)
	defer c.free(namePtr)

	handlePtr := c.newBytes(4)
	defer c.free(handlePtr)

	var vfsPtr uint32
	if vfs != "" {
		vfsPtr = c.newString(vfs)
		defer c.free(vfsPtr)
	}

	r, err := c.api.open.Call(ctx, uint64(namePtr), uint64(handlePtr), flags, uint64(vfsPtr))
	if err != nil {
		_ = c.Close()
		return nil, err
	}

	c.handle, _ = c.memory.ReadUint32Le(handlePtr)

	if r[0] != SQLITE_OK {
		err := fmt.Errorf("sqlite error (%d): %s", r[0], c.Errmsg())
		_ = c.Close()
		return nil, err
	}
	return &c, nil
}

func (c *Conn) Errmsg() error {
	r, err := c.api.errmsg.Call(context.TODO(), uint64(c.handle))
	if err != nil {
		return err
	}
	return errors.New(c.getString(uint32(r[0]), 64))
}

func (c *Conn) Close() error {
	r, err := c.api.close.Call(context.TODO(), uint64(c.handle))
	if err != nil {
		return err
	}

	if r[0] != SQLITE_OK {
		return fmt.Errorf("sqlite error (%d): %s", r[0], c.Errmsg())
	}
	return nil
}

func (c *Conn) free(ptr uint32) {
	_, err := c.api.free.Call(context.TODO(), uint64(ptr))
	if err != nil {
		panic(err)
	}
}

func (c *Conn) newBytes(len uint32) uint32 {
	r, err := c.api.malloc.Call(context.TODO(), uint64(len))
	if err != nil {
		panic(err)
	}
	if r[0] == 0 {
		panic("sqlite3: out of memory")
	}
	return uint32(r[0])
}

func (c *Conn) newString(str string) uint32 {
	ptr := c.newBytes(uint32(len(str) + 1))

	buf, ok := c.memory.Read(ptr, uint32(len(str)+1))
	if !ok {
		c.api.free.Call(context.TODO(), uint64(ptr))
		panic("sqlite3: failed to init string")
	}

	buf[len(str)] = 0
	copy(buf, str)
	return ptr
}

func (c *Conn) getString(ptr, maxlen uint32) string {
	buf, ok := c.memory.Read(ptr, maxlen)
	if !ok {
		if size := c.memory.Size(); ptr < size {
			buf, ok = c.memory.Read(ptr, size-ptr)
		}
		if !ok {
			panic("sqlite3: invalid pointer")
		}
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic("sqlite3: missing NUL terminator")
	} else {
		return string(buf[:i])
	}
}

const (
	SQLITE_OK   = 0
	SQLITE_ROW  = 100
	SQLITE_DONE = 101

	SQLITE_OPEN_READWRITE = 0x00000002
	SQLITE_OPEN_CREATE    = 0x00000004
)

type sqliteAPI struct {
	malloc      api.Function
	free        api.Function
	errmsg      api.Function
	open        api.Function
	close       api.Function
	prepare     api.Function
	exec        api.Function
	step        api.Function
	columnInt   api.Function
	columnText  api.Function
	columnFloat api.Function
}
