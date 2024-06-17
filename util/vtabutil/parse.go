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
	code = 4
	base = 8
)

var (
	//go:embed parse/sql3parse_table.wasm
	binary  []byte
	ctx     context.Context
	once    sync.Once
	runtime wazero.Runtime
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
func Parse(sql string) (*Table, error) {
	once.Do(func() {
		ctx = context.Background()
		cfg := wazero.NewRuntimeConfigInterpreter().WithDebugInfoEnabled(false)
		runtime = wazero.NewRuntimeWithConfig(ctx, cfg)
	})

	mod, err := runtime.InstantiateWithConfig(ctx, binary, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, err
	}

	if buf, ok := mod.Memory().Read(base, uint32(len(sql))); ok {
		copy(buf, sql)
	}
	r, err := mod.ExportedFunction("sql3parse_table").Call(ctx, base, uint64(len(sql)), code)
	if err != nil {
		return nil, err
	}

	c, _ := mod.Memory().ReadUint32Le(code)
	if c == uint32(_MEMORY) {
		panic(util.OOMErr)
	}
	if c != uint32(_NONE) {
		return nil, ecode(c)
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
	if r[0] == 0 {
		return ""
	}
	off, _ := c.tab.mod.Memory().ReadUint32Le(uint32(r[0]) + 0)
	len, _ := c.tab.mod.Memory().ReadUint32Le(uint32(r[0]) + 4)
	return c.tab.sql[off-base : off+len-base]
}

type ecode uint32

const (
	_NONE ecode = iota
	_MEMORY
	_SYNTAX
	_UNSUPPORTEDSQL
)

func (e ecode) Error() string {
	switch e {
	case _SYNTAX:
		return "sql3parse: invalid syntax"
	case _UNSUPPORTEDSQL:
		return "sql3parse: unsupported SQL"
	default:
		panic(util.AssertErr())
	}
}
