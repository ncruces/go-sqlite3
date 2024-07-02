package vtabutil

import (
	"context"
	"sync"
	"unsafe"

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

	tab := sql3ptr[sql3table, Table](r[0]).value(sql, mod)
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

type sql3table struct {
	name            sql3string
	schema          sql3string
	comment         sql3string
	is_temporary    bool
	is_ifnotexists  bool
	is_withoutrowid bool
	is_strict       bool
	num_columns     int32
	columns         uint32
	num_constraint  int32
	constraints     uint32
	typ             sql3statement_type
	current_name    sql3string
	new_name        sql3string
}

func (s sql3table) value(sql string, mod api.Module) (r Table) {
	r.Name = s.name.value(sql, nil)
	r.Schema = s.schema.value(sql, nil)
	r.Comment = s.comment.value(sql, nil)

	r.IsTemporary = s.is_temporary
	r.IsIfNotExists = s.is_ifnotexists
	r.IsWithoutRowID = s.is_withoutrowid
	r.IsStrict = s.is_strict

	r.Columns = make([]Column, s.num_columns)
	ptr, _ := mod.Memory().ReadUint32Le(s.columns)
	col := sql3ptr[sql3column, Column](ptr)
	for i := range r.Columns {
		r.Columns[i] = col.value(sql, mod)
		col += 4
	}

	r.CurrentName = s.current_name.value(sql, nil)
	r.NewName = s.new_name.value(sql, nil)

	return
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

type sql3column struct {
	name                   sql3string
	typ                    sql3string
	length                 sql3string
	constraint_name        sql3string
	comment                sql3string
	is_primarykey          bool
	is_autoincrement       bool
	is_notnull             bool
	is_unique              bool
	pk_order               sql3order_clause
	pk_conflictclause      sql3conflict_clause
	notnull_conflictclause sql3conflict_clause
	unique_conflictclause  sql3conflict_clause
	check_expr             sql3string
	default_expr           sql3string
	collate_name           sql3string
	foreignkey_clause      sql3foreignkey
}

func (s sql3column) value(sql string, _ api.Module) (r Column) {
	r.Name = s.name.value(sql, nil)
	r.Type = s.typ.value(sql, nil)
	r.Length = s.length.value(sql, nil)
	r.ConstraintName = s.constraint_name.value(sql, nil)
	r.Comment = s.comment.value(sql, nil)

	r.IsPrimaryKey = s.is_primarykey
	r.IsAutoIncrement = s.is_autoincrement
	r.IsNotNull = s.is_notnull
	r.IsUnique = s.is_unique

	r.CheckExpr = s.check_expr.value(sql, nil)
	r.DefaultExpr = s.default_expr.value(sql, nil)
	r.CollateName = s.collate_name.value(sql, nil)

	return
}

type sql3foreignkey struct {
	table       sql3string
	num_columns int32
	column_name uint32
	on_delete   sql3fk_action
	on_update   sql3fk_action
	match       sql3string
	deferrable  sql3fk_deftype
}

type sql3string struct {
	off uint32
	len uint32
}

func (s sql3string) value(sql string, _ api.Module) string {
	if s.off == 0 {
		return ""
	}
	return sql[s.off-sqlp : s.off+s.len-sqlp]
}

type sql3ptr[T sql3valuer[V], V any] uint32

func (s sql3ptr[T, V]) value(sql string, mod api.Module) (_ V) {
	if s == 0 {
		return
	}
	var val T
	buf, _ := mod.Memory().Read(uint32(s), uint32(unsafe.Sizeof(val)))
	val = *(*T)(unsafe.Pointer(&buf[0]))
	return val.value(sql, mod)
}

type sql3valuer[T any] interface {
	value(string, api.Module) T
}

type (
	sql3conflict_clause int32
	sql3order_clause    int32
	sql3fk_action       int32
	sql3fk_deftype      int32
	sql3statement_type  int32
)
