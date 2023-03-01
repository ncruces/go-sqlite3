// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

func newConn(ctx context.Context, module api.Module) (_ *Conn, err error) {
	getFun := func(name string) api.Function {
		f := module.ExportedFunction(name)
		if f == nil {
			err = noFuncErr + errorString(name)
			return nil
		}
		return f
	}

	getVal := func(name string) uint32 {
		global := module.ExportedGlobal(name)
		if global == nil {
			err = noGlobalErr + errorString(name)
			return 0
		}
		return memory{module}.readUint32(uint32(global.Get()))
	}

	c := Conn{
		ctx: ctx,
		mem: memory{module},
		api: sqliteAPI{
			free:          getFun("free"),
			malloc:        getFun("malloc"),
			destructor:    uint64(getVal("malloc_destructor")),
			errcode:       getFun("sqlite3_errcode"),
			errstr:        getFun("sqlite3_errstr"),
			errmsg:        getFun("sqlite3_errmsg"),
			erroff:        getFun("sqlite3_error_offset"),
			open:          getFun("sqlite3_open_v2"),
			close:         getFun("sqlite3_close"),
			prepare:       getFun("sqlite3_prepare_v3"),
			finalize:      getFun("sqlite3_finalize"),
			reset:         getFun("sqlite3_reset"),
			step:          getFun("sqlite3_step"),
			exec:          getFun("sqlite3_exec"),
			clearBindings: getFun("sqlite3_clear_bindings"),
			bindCount:     getFun("sqlite3_bind_parameter_count"),
			bindIndex:     getFun("sqlite3_bind_parameter_index"),
			bindName:      getFun("sqlite3_bind_parameter_name"),
			bindNull:      getFun("sqlite3_bind_null"),
			bindInteger:   getFun("sqlite3_bind_int64"),
			bindFloat:     getFun("sqlite3_bind_double"),
			bindText:      getFun("sqlite3_bind_text64"),
			bindBlob:      getFun("sqlite3_bind_blob64"),
			bindZeroBlob:  getFun("sqlite3_bind_zeroblob64"),
			columnCount:   getFun("sqlite3_column_count"),
			columnName:    getFun("sqlite3_column_name"),
			columnType:    getFun("sqlite3_column_type"),
			columnInteger: getFun("sqlite3_column_int64"),
			columnFloat:   getFun("sqlite3_column_double"),
			columnText:    getFun("sqlite3_column_text"),
			columnBlob:    getFun("sqlite3_column_blob"),
			columnBytes:   getFun("sqlite3_column_bytes"),
			autocommit:    getFun("sqlite3_get_autocommit"),
			lastRowid:     getFun("sqlite3_last_insert_rowid"),
			changes:       getFun("sqlite3_changes64"),
			blobOpen:      getFun("sqlite3_blob_open"),
			blobClose:     getFun("sqlite3_blob_close"),
			blobReopen:    getFun("sqlite3_blob_reopen"),
			blobBytes:     getFun("sqlite3_blob_bytes"),
			blobRead:      getFun("sqlite3_blob_read"),
			blobWrite:     getFun("sqlite3_blob_write"),
			interrupt:     getVal("sqlite3_interrupt_offset"),
		},
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

type sqliteAPI struct {
	free          api.Function
	malloc        api.Function
	destructor    uint64
	errcode       api.Function
	errstr        api.Function
	errmsg        api.Function
	erroff        api.Function
	open          api.Function
	close         api.Function
	prepare       api.Function
	finalize      api.Function
	reset         api.Function
	step          api.Function
	exec          api.Function
	clearBindings api.Function
	bindNull      api.Function
	bindCount     api.Function
	bindIndex     api.Function
	bindName      api.Function
	bindInteger   api.Function
	bindFloat     api.Function
	bindText      api.Function
	bindBlob      api.Function
	bindZeroBlob  api.Function
	columnCount   api.Function
	columnName    api.Function
	columnType    api.Function
	columnInteger api.Function
	columnFloat   api.Function
	columnText    api.Function
	columnBlob    api.Function
	columnBytes   api.Function
	autocommit    api.Function
	lastRowid     api.Function
	changes       api.Function
	blobOpen      api.Function
	blobClose     api.Function
	blobReopen    api.Function
	blobBytes     api.Function
	blobRead      api.Function
	blobWrite     api.Function
	interrupt     uint32
}
