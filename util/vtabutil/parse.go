package vtabutil

import (
	"context"
	"sync"

	_ "embed"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	_NONE = iota
	_MEMORY
	_SYNTAX
	_UNSUPPORTEDSQL

	codeptr = 4
	baseptr = 8
)

var (
	//go:embed parse/sql3parse_table.wasm
	binary  []byte
	ctx     context.Context
	once    sync.Once
	runtime wazero.Runtime
	module  wazero.CompiledModule
)

// Table holds metadata about a table.
type Table struct {
	mod api.Module
	ptr uint32
	sql string
}

// Parse parses a [CREATE] or [ALTER TABLE] command.
//
// [CREATE]: https://sqlite.org/lang_createtable.html
// [ALTER TABLE]: https://sqlite.org/lang_altertable.html
func Parse(sql string) (_ *Table, err error) {
	once.Do(func() {
		ctx = context.Background()
		cfg := wazero.NewRuntimeConfigInterpreter().WithDebugInfoEnabled(false)
		runtime = wazero.NewRuntimeWithConfig(ctx, cfg)
		module, err = runtime.CompileModule(ctx, binary)
	})
	if err != nil {
		return nil, err
	}

	mod, err := runtime.InstantiateModule(ctx, module, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, err
	}

	if buf, ok := mod.Memory().Read(baseptr, uint32(len(sql))); ok {
		copy(buf, sql)
	}
	r, err := mod.ExportedFunction("sql3parse_table").Call(ctx, baseptr, uint64(len(sql)), codeptr)
	if err != nil {
		return nil, err
	}

	c, _ := mod.Memory().ReadUint32Le(codeptr)
	switch c {
	case _MEMORY:
		panic(util.OOMErr)
	case _SYNTAX:
		return nil, util.ErrorString("sql3parse: invalid syntax")
	case _UNSUPPORTEDSQL:
		return nil, util.ErrorString("sql3parse: unsupported SQL")
	}
	if r[0] == 0 {
		return nil, nil
	}
	return &Table{
		sql: sql,
		mod: mod,
		ptr: uint32(r[0]),
	}, nil
}

// Close closes a table handle.
func (t *Table) Close() error {
	mod := t.mod
	t.mod = nil
	return mod.Close(ctx)
}

// NumColumns returns the number of columns of the table.
func (t *Table) NumColumns() int {
	r, err := t.mod.ExportedFunction("sql3table_num_columns").Call(ctx, uint64(t.ptr))
	if err != nil {
		panic(err)
	}
	return int(int32(r[0]))
}

// Column returns data for the ith column of the table.
//
// https://sqlite.org/lang_createtable.html#column_definitions
func (t *Table) Column(i int) Column {
	r, err := t.mod.ExportedFunction("sql3table_get_column").Call(ctx, uint64(t.ptr), uint64(i))
	if err != nil {
		panic(err)
	}
	return Column{
		tab: t,
		ptr: uint32(r[0]),
	}
}

func (t *Table) string(ptr uint32) string {
	if ptr == 0 {
		return ""
	}
	off, _ := t.mod.Memory().ReadUint32Le(ptr + 0)
	len, _ := t.mod.Memory().ReadUint32Le(ptr + 4)
	return t.sql[off-baseptr : off+len-baseptr]
}

// Column holds metadata about a column.
type Column struct {
	tab *Table
	ptr uint32
}

// Type returns the declared type of a column.
//
// https://sqlite.org/lang_createtable.html#column_data_types
func (c Column) Type() string {
	r, err := c.tab.mod.ExportedFunction("sql3column_type").Call(ctx, uint64(c.ptr))
	if err != nil {
		panic(err)
	}
	return c.tab.string(uint32(r[0]))
}
