// Package statement defines virtual tables and table-valued functions natively using SQL.
//
// https://github.com/0x09/sqlite-statement-vtab
package statement

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/ncruces/go-sqlite3"
)

// Register registers the statement virtual table.
func Register(db *sqlite3.Conn) {
	declare := func(db *sqlite3.Conn, _, _, _ string, arg ...string) (_ *table, err error) {
		if len(arg) == 0 || len(arg[0]) < 3 {
			return nil, fmt.Errorf("statement: no statement provided")
		}
		sql := arg[0]
		if len := len(sql); sql[0] != '(' || sql[len-1] != ')' {
			return nil, fmt.Errorf("statement: statement must be parenthesized")
		} else {
			sql = sql[1 : len-1]
		}

		table := &table{sql: sql}
		err = table.declare(db)
		if err != nil {
			table.Close()
			return nil, err
		}
		return table, nil
	}

	sqlite3.CreateModule(db, "statement", declare, declare)
}

type table struct {
	stmt  *sqlite3.Stmt
	sql   string
	inuse bool
}

func (t *table) declare(db *sqlite3.Conn) (err error) {
	var tail string
	t.stmt, tail, err = db.Prepare(t.sql)
	if err != nil {
		return err
	}
	if tail != "" {
		return fmt.Errorf("statement: multiple statements")
	}
	if !t.stmt.ReadOnly() {
		return fmt.Errorf("statement: statement must be read only")
	}

	var sep = ""
	var str strings.Builder
	str.WriteString(`CREATE TABLE x(`)
	outputs := t.stmt.ColumnCount()
	for i := 0; i < outputs; i++ {
		str.WriteString(sep)
		name := t.stmt.ColumnName(i)
		str.WriteString(sqlite3.QuoteIdentifier(name))
		str.WriteByte(' ')
		str.WriteString(t.stmt.ColumnDeclType(i))
		sep = ","
	}
	inputs := t.stmt.BindCount()
	for i := 1; i <= inputs; i++ {
		str.WriteString(sep)
		name := t.stmt.BindName(i)
		if name == "" {
			str.WriteString("[")
			str.WriteString(strconv.Itoa(i))
			str.WriteString("] HIDDEN")
		} else {
			str.WriteString(sqlite3.QuoteIdentifier(name[1:]))
			str.WriteString(" HIDDEN")
		}
		sep = ","
	}
	str.WriteByte(')')
	return db.DeclareVtab(str.String())
}

func (t *table) Close() error {
	return t.stmt.Close()
}

func (t *table) BestIndex(idx *sqlite3.IndexInfo) error {
	idx.EstimatedCost = 1000

	var argvIndex = 1
	var needIndex bool
	var listIndex []int
	outputs := t.stmt.ColumnCount()
	for i, cst := range idx.Constraint {
		// Skip if this is a constraint on one of our output columns.
		if cst.Column < outputs {
			continue
		}

		// A given query plan is only usable if all provided input columns
		// are usable and have equal constraints only.
		if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
			return sqlite3.CONSTRAINT
		}

		// The non-zero argvIdx values must be contiguous.
		// If they're not, build a list and serialize it through IdxStr.
		nextIndex := cst.Column - outputs + 1
		idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
			ArgvIndex: argvIndex,
			Omit:      true,
		}
		if nextIndex != argvIndex {
			needIndex = true
		}
		listIndex = append(listIndex, nextIndex)
		argvIndex++
	}

	if needIndex {
		buf, err := json.Marshal(listIndex)
		if err != nil {
			return err
		}
		idx.IdxStr = unsafe.String(&buf[0], len(buf))
	}
	return nil
}

func (t *table) Open() (_ sqlite3.VTabCursor, err error) {
	stmt := t.stmt
	if !t.inuse {
		t.inuse = true
	} else {
		stmt, _, err = t.stmt.Conn().Prepare(t.sql)
		if err != nil {
			return nil, err
		}
	}
	return &cursor{table: t, stmt: stmt}, nil
}

func (t *table) Rename(new string) error {
	return nil
}

type cursor struct {
	table *table
	stmt  *sqlite3.Stmt
	arg   []sqlite3.Value
	rowID int64
}

func (c *cursor) Close() error {
	if c.stmt == c.table.stmt {
		c.table.inuse = false
		c.stmt.ClearBindings()
		return c.stmt.Reset()
	}
	return c.stmt.Close()
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	c.arg = arg
	c.rowID = 0
	c.stmt.ClearBindings()
	if err := c.stmt.Reset(); err != nil {
		return err
	}

	var list []int
	if idxStr != "" {
		buf := unsafe.Slice(unsafe.StringData(idxStr), len(idxStr))
		err := json.Unmarshal(buf, &list)
		if err != nil {
			return err
		}
	}

	for i, arg := range arg {
		param := i + 1
		if list != nil {
			param = list[i]
		}
		err := c.stmt.BindValue(param, arg)
		if err != nil {
			return err
		}
	}
	return c.Next()
}

func (c *cursor) Next() error {
	if c.stmt.Step() {
		c.rowID++
		return nil
	}
	return c.stmt.Err()
}

func (c *cursor) EOF() bool {
	return !c.stmt.Busy()
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx *sqlite3.Context, col int) error {
	switch outputs := c.stmt.ColumnCount(); {
	case col < outputs:
		ctx.ResultValue(c.stmt.ColumnValue(col))
	case col-outputs < len(c.arg):
		ctx.ResultValue(c.arg[col-outputs])
	}
	return nil
}
