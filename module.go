// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"
	"crypto/rand"
	"math"
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

var sqlite3 struct {
	once      sync.Once
	runtime   wazero.Runtime
	compiled  wazero.CompiledModule
	instances atomic.Uint64
	err       error
}

func instantiateModule() (*module, error) {
	ctx := context.Background()

	sqlite3.once.Do(compileModule)
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
	return newModule(mod)
}

func compileModule() {
	ctx := context.Background()
	sqlite3.runtime = wazero.NewRuntime(ctx)
	vfsInstantiate(ctx, sqlite3.runtime)

	bin := Binary
	if bin == nil && Path != "" {
		bin, sqlite3.err = os.ReadFile(Path)
		if sqlite3.err != nil {
			return
		}
	}
	if bin == nil {
		sqlite3.err = binaryErr
		return
	}

	sqlite3.compiled, sqlite3.err = sqlite3.runtime.CompileModule(ctx, bin)
}

type module struct {
	api.Module

	ctx context.Context
	mem memory
	api sqliteAPI
}

func newModule(mod api.Module) (m *module, err error) {
	getFun := func(name string) api.Function {
		f := m.ExportedFunction(name)
		if f == nil {
			err = noFuncErr + errorString(name)
			return nil
		}
		return f
	}

	getVal := func(name string) uint32 {
		global := m.ExportedGlobal(name)
		if global == nil {
			err = noGlobalErr + errorString(name)
			return 0
		}
		return m.mem.readUint32(uint32(global.Get()))
	}

	m = &module{
		Module: mod,
		mem:    memory{mod},
		ctx:    context.Background(),
	}
	m.api = sqliteAPI{
		free:            getFun("free"),
		malloc:          getFun("malloc"),
		destructor:      uint64(getVal("malloc_destructor")),
		errcode:         getFun("sqlite3_errcode"),
		errstr:          getFun("sqlite3_errstr"),
		errmsg:          getFun("sqlite3_errmsg"),
		erroff:          getFun("sqlite3_error_offset"),
		open:            getFun("sqlite3_open_v2"),
		close:           getFun("sqlite3_close"),
		closeZombie:     getFun("sqlite3_close_v2"),
		prepare:         getFun("sqlite3_prepare_v3"),
		finalize:        getFun("sqlite3_finalize"),
		reset:           getFun("sqlite3_reset"),
		step:            getFun("sqlite3_step"),
		exec:            getFun("sqlite3_exec"),
		clearBindings:   getFun("sqlite3_clear_bindings"),
		bindCount:       getFun("sqlite3_bind_parameter_count"),
		bindIndex:       getFun("sqlite3_bind_parameter_index"),
		bindName:        getFun("sqlite3_bind_parameter_name"),
		bindNull:        getFun("sqlite3_bind_null"),
		bindInteger:     getFun("sqlite3_bind_int64"),
		bindFloat:       getFun("sqlite3_bind_double"),
		bindText:        getFun("sqlite3_bind_text64"),
		bindBlob:        getFun("sqlite3_bind_blob64"),
		bindZeroBlob:    getFun("sqlite3_bind_zeroblob64"),
		columnCount:     getFun("sqlite3_column_count"),
		columnName:      getFun("sqlite3_column_name"),
		columnType:      getFun("sqlite3_column_type"),
		columnInteger:   getFun("sqlite3_column_int64"),
		columnFloat:     getFun("sqlite3_column_double"),
		columnText:      getFun("sqlite3_column_text"),
		columnBlob:      getFun("sqlite3_column_blob"),
		columnBytes:     getFun("sqlite3_column_bytes"),
		autocommit:      getFun("sqlite3_get_autocommit"),
		lastRowid:       getFun("sqlite3_last_insert_rowid"),
		changes:         getFun("sqlite3_changes64"),
		blobOpen:        getFun("sqlite3_blob_open"),
		blobClose:       getFun("sqlite3_blob_close"),
		blobReopen:      getFun("sqlite3_blob_reopen"),
		blobBytes:       getFun("sqlite3_blob_bytes"),
		blobRead:        getFun("sqlite3_blob_read"),
		blobWrite:       getFun("sqlite3_blob_write"),
		backupInit:      getFun("sqlite3_backup_init"),
		backupStep:      getFun("sqlite3_backup_step"),
		backupFinish:    getFun("sqlite3_backup_finish"),
		backupRemaining: getFun("sqlite3_backup_remaining"),
		backupPageCount: getFun("sqlite3_backup_pagecount"),
		interrupt:       getVal("sqlite3_interrupt_offset"),
	}
	if err != nil {
		m = nil
	}
	return
}

func (m *module) error(rc uint64, handle uint32, sql ...string) error {
	if rc == _OK {
		return nil
	}

	err := Error{code: rc}

	if err.Code() == NOMEM || err.ExtendedCode() == IOERR_NOMEM {
		panic(oomErr)
	}

	var r []uint64

	r, _ = m.api.errstr.Call(m.ctx, rc)
	if r != nil {
		err.str = m.mem.readString(uint32(r[0]), _MAX_STRING)
	}

	r, _ = m.api.errmsg.Call(m.ctx, uint64(handle))
	if r != nil {
		err.msg = m.mem.readString(uint32(r[0]), _MAX_STRING)
	}

	if sql != nil {
		r, _ = m.api.erroff.Call(m.ctx, uint64(handle))
		if r != nil && r[0] != math.MaxUint32 {
			err.sql = sql[0][r[0]:]
		}
	}

	switch err.msg {
	case err.str, "not an error":
		err.msg = ""
	}
	return &err
}

type sqliteAPI struct {
	free            api.Function
	malloc          api.Function
	destructor      uint64
	errcode         api.Function
	errstr          api.Function
	errmsg          api.Function
	erroff          api.Function
	open            api.Function
	close           api.Function
	closeZombie     api.Function
	prepare         api.Function
	finalize        api.Function
	reset           api.Function
	step            api.Function
	exec            api.Function
	clearBindings   api.Function
	bindNull        api.Function
	bindCount       api.Function
	bindIndex       api.Function
	bindName        api.Function
	bindInteger     api.Function
	bindFloat       api.Function
	bindText        api.Function
	bindBlob        api.Function
	bindZeroBlob    api.Function
	columnCount     api.Function
	columnName      api.Function
	columnType      api.Function
	columnInteger   api.Function
	columnFloat     api.Function
	columnText      api.Function
	columnBlob      api.Function
	columnBytes     api.Function
	autocommit      api.Function
	lastRowid       api.Function
	changes         api.Function
	blobOpen        api.Function
	blobClose       api.Function
	blobReopen      api.Function
	blobBytes       api.Function
	blobRead        api.Function
	blobWrite       api.Function
	backupInit      api.Function
	backupStep      api.Function
	backupFinish    api.Function
	backupRemaining api.Function
	backupPageCount api.Function
	interrupt       uint32
}
