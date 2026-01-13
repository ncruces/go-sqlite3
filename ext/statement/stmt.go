// Package statement defines table-valued functions using SQL.
//
// It can be used to create "parametrized views":
// pre-packaged queries that can be parametrized at query execution time.
//
// https://github.com/0x09/sqlite-statement-vtab
package statement

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unsafe"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Register registers the statement virtual table.
func Register(db *sqlite3.Conn) error {
	return sqlite3.CreateModule(db, "statement", declare, declare)
}

type table struct {
	stmt  *sqlite3.Stmt
	sql   string
	inuse bool
}

func declare(db *sqlite3.Conn, _, _, _ string, arg ...string) (*table, error) {
	if len(arg) != 1 {
		return nil, util.ErrorString("statement: wrong number of arguments")
	}

	sql := "SELECT * FROM\n" + arg[0]

	stmt, tail, err := db.PrepareFlags(sql, sqlite3.PREPARE_PERSISTENT|sqlite3.PREPARE_FROM_DDL)
	if err != nil {
		return nil, err
	}
	if tail != "" {
		return nil, util.TailErr
	}

	var sep string
	var str strings.Builder
	str.WriteString("CREATE TABLE x(")
	outputs := stmt.ColumnCount()
	for i := range outputs {
		name := sqlite3.QuoteIdentifier(stmt.ColumnName(i))
		str.WriteString(sep)
		str.WriteString(name)
		str.WriteString(" ")
		str.WriteString(stmt.ColumnDeclType(i))
		sep = ","
	}
	inputs := stmt.BindCount()
	for i := 1; i <= inputs; i++ {
		str.WriteString(sep)
		name := stmt.BindName(i)
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

	err = db.DeclareVTab(str.String())
	if err != nil {
		stmt.Close()
		return nil, err
	}

	return &table{sql: sql, stmt: stmt}, nil
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
		stmt, _, err = t.stmt.Conn().PrepareFlags(t.sql, sqlite3.PREPARE_FROM_DDL)
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
		return errors.Join(
			c.stmt.Reset(),
			c.stmt.ClearBindings())
	}
	return c.stmt.Close()
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	err := errors.Join(
		c.stmt.Reset(),
		c.stmt.ClearBindings())
	if err != nil {
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
	c.arg = append(c.arg[:0], arg...)
	c.rowID = 0
	return c.Next()
}

func (c *cursor) Next() error {
	if c.stmt.Step() {
		c.rowID++
	}
	return c.stmt.Err()
}

func (c *cursor) EOF() bool {
	return !c.stmt.Busy()
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx sqlite3.Context, col int) error {
	switch outputs := c.stmt.ColumnCount(); {
	case col < outputs:
		ctx.ResultValue(c.stmt.ColumnValue(col))
	case col-outputs < len(c.arg):
		ctx.ResultValue(c.arg[col-outputs])
	}
	return nil
}
