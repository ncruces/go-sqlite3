package sqlite3

import "github.com/tetratelabs/wazero/api"

func newConn(module api.Module) *Conn {
	getFun := func(name string) api.Function {
		f := module.ExportedFunction(name)
		if f == nil {
			panic(noFuncErr + errorString(name))
		}
		return f
	}

	global := module.ExportedGlobal("malloc_destructor")
	if global == nil {
		panic(noGlobalErr + "malloc_destructor")
	}
	destructor := uint32(global.Get())
	destructor, ok := module.Memory().ReadUint32Le(destructor)
	if !ok {
		panic(noGlobalErr + "malloc_destructor")
	}

	return &Conn{
		module: module,
		memory: module.Memory(),
		api: sqliteAPI{
			malloc:        getFun("malloc"),
			free:          getFun("free"),
			destructor:    uint64(destructor),
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
			bindInteger:   getFun("sqlite3_bind_int64"),
			bindFloat:     getFun("sqlite3_bind_double"),
			bindText:      getFun("sqlite3_bind_text64"),
			bindBlob:      getFun("sqlite3_bind_blob64"),
			bindZeroBlob:  getFun("sqlite3_bind_zeroblob64"),
			bindNull:      getFun("sqlite3_bind_null"),
			columnInteger: getFun("sqlite3_column_int64"),
			columnFloat:   getFun("sqlite3_column_double"),
			columnText:    getFun("sqlite3_column_text"),
			columnBlob:    getFun("sqlite3_column_blob"),
			columnBytes:   getFun("sqlite3_column_bytes"),
			columnType:    getFun("sqlite3_column_type"),
		},
	}
}

type sqliteAPI struct {
	malloc        api.Function
	free          api.Function
	destructor    uint64
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
	bindInteger   api.Function
	bindFloat     api.Function
	bindText      api.Function
	bindBlob      api.Function
	bindZeroBlob  api.Function
	bindNull      api.Function
	columnInteger api.Function
	columnFloat   api.Function
	columnText    api.Function
	columnBlob    api.Function
	columnBytes   api.Function
	columnType    api.Function
}
