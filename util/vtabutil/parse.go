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

	errp = 4
	sqlp = 8
)

var (
	//go:embed parse/sql3parse_table.wasm
	binary   []byte
	once     sync.Once
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
)

// Parse parses a [CREATE] or [ALTER TABLE] command.
//
// [CREATE]: https://sqlite.org/lang_createtable.html
// [ALTER TABLE]: https://sqlite.org/lang_altertable.html
func Parse(sql string) (_ *Table, err error) {
	once.Do(func() {
		ctx := context.Background()
		cfg := wazero.NewRuntimeConfigInterpreter()
		runtime = wazero.NewRuntimeWithConfig(ctx, cfg)
		compiled, err = runtime.CompileModule(ctx, binary)
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	mod, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, err
	}
	defer mod.Close(ctx)

	if buf, ok := mod.Memory().Read(sqlp, uint32(len(sql))); ok {
		copy(buf, sql)
	}

	r, err := mod.ExportedFunction("sql3parse_table").Call(ctx, sqlp, uint64(len(sql)), errp)
	if err != nil {
		return nil, err
	}

	c, _ := mod.Memory().ReadUint32Le(errp)
	switch c {
	case _MEMORY:
		panic(util.OOMErr)
	case _SYNTAX:
		return nil, util.ErrorString("sql3parse: invalid syntax")
	case _UNSUPPORTEDSQL:
		return nil, util.ErrorString("sql3parse: unsupported SQL")
	}

	var tab Table
	tab.load(mod, uint32(r[0]), sql)
	return &tab, nil
}

// Table holds metadata about a table.
type Table struct {
	Name           string
	Schema         string
	Comment        string
	IsTemporary    bool
	IsIfNotExists  bool
	IsWithoutRowID bool
	IsStrict       bool
	Columns        []Column
	CurrentName    string
	NewName        string
}

func (t *Table) load(mod api.Module, ptr uint32, sql string) {
	t.Name = loadString(mod, ptr+0, sql)
	t.Schema = loadString(mod, ptr+8, sql)
	t.Comment = loadString(mod, ptr+16, sql)

	t.IsTemporary = loadBool(mod, ptr+24)
	t.IsIfNotExists = loadBool(mod, ptr+25)
	t.IsWithoutRowID = loadBool(mod, ptr+26)
	t.IsStrict = loadBool(mod, ptr+27)

	num, _ := mod.Memory().ReadUint32Le(ptr + 28)
	t.Columns = make([]Column, num)
	ref, _ := mod.Memory().ReadUint32Le(ptr + 32)
	for i := range t.Columns {
		p, _ := mod.Memory().ReadUint32Le(ref)
		t.Columns[i].load(mod, p, sql)
		ref += 4
	}

	t.CurrentName = loadString(mod, ptr+48, sql)
	t.NewName = loadString(mod, ptr+56, sql)
}

// Column holds metadata about a column.
type Column struct {
	Name            string
	Type            string
	Length          string
	ConstraintName  string
	Comment         string
	IsPrimaryKey    bool
	IsAutoIncrement bool
	IsNotNull       bool
	IsUnique        bool
	CheckExpr       string
	DefaultExpr     string
	CollateName     string
}

func (c *Column) load(mod api.Module, ptr uint32, sql string) {
	c.Name = loadString(mod, ptr+0, sql)
	c.Type = loadString(mod, ptr+8, sql)
	c.Length = loadString(mod, ptr+16, sql)
	c.ConstraintName = loadString(mod, ptr+24, sql)
	c.Comment = loadString(mod, ptr+32, sql)

	c.IsPrimaryKey = loadBool(mod, ptr+40)
	c.IsAutoIncrement = loadBool(mod, ptr+41)
	c.IsNotNull = loadBool(mod, ptr+42)
	c.IsUnique = loadBool(mod, ptr+43)

	c.CheckExpr = loadString(mod, ptr+60, sql)
	c.DefaultExpr = loadString(mod, ptr+68, sql)
	c.CollateName = loadString(mod, ptr+76, sql)
}

func loadString(mod api.Module, ptr uint32, sql string) string {
	off, _ := mod.Memory().ReadUint32Le(ptr + 0)
	if off == 0 {
		return ""
	}
	len, _ := mod.Memory().ReadUint32Le(ptr + 4)
	return sql[off-sqlp : off+len-sqlp]
}

func loadBool(mod api.Module, ptr uint32) bool {
	val, _ := mod.Memory().ReadByte(ptr)
	return val != 0
}
