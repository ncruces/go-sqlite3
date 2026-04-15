package sqlite3

import "github.com/ncruces/go-sqlite3/internal/errutil"

// ExtensionLibrary represents a dynamically linked SQLite extension.
type ExtensionLibrary interface {
	Xsqlite3_extension_init(db, _, _ int32) int32
}

// ExtensionInfo returns values needed to load a dynamically linked SQLite extension.
type ExtensionInfo func() (memorySize, memoryAlignment, tableSize, tableAlignment int64)

type extEnv struct {
	*env
	memoryBase int32
	tableBase  int32
}

func (e *extEnv) X__memory_base() *int32 { return &e.memoryBase }
func (e *extEnv) X__table_base() *int32  { return &e.tableBase }

// ExtensionInit loads an SQLite extension library.
//
// https://sqlite.org/loadext.html
func ExtensionInit[Env any, Mod ExtensionLibrary](db *Conn, init func(env Env) Mod, info ExtensionInfo) error {
	memSize, memAlign, tableSize, tableAlign := info()

	var memBase int32
	if memSize > 0 {
		memBase = db.wrp.Xaligned_alloc(int32(memAlign), int32(memSize))
		if memBase == 0 {
			panic(errutil.OOMErr)
		}
	}

	var tableBase int
	if tableSize > 0 {
		// Round up to the alignment.
		rnd := int(tableAlign) - 1
		tab := db.wrp.X__indirect_function_table()
		tableBase = (len(*tab) + rnd) &^ rnd
		if add := tableBase + int(tableSize) - len(*tab); add > 0 {
			*tab = append(*tab, make([]any, add)...)
		}
	}

	e := &extEnv{
		env:        &env{db.wrp},
		memoryBase: memBase,
		tableBase:  int32(tableBase),
	}

	mod := init(any(e).(Env))
	if opt, ok := any(mod).(interface{ X__wasm_apply_data_relocs() }); ok {
		opt.X__wasm_apply_data_relocs()
	}
	if opt, ok := any(mod).(interface{ X__wasm_call_ctors() }); ok {
		opt.X__wasm_call_ctors()
	}
	rc := mod.Xsqlite3_extension_init(int32(db.handle), 0, 0)
	return db.error(res_t(rc))
}
