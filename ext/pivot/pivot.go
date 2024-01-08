// Package pivot implements a pivot virtual table.
//
// https://github.com/jakethaw/pivot_vtab
package pivot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

// Register registers the pivot virtual table.
func Register(db *sqlite3.Conn) {
	sqlite3.CreateModule(db, "pivot", declare, declare)
}

type table struct {
	db   *sqlite3.Conn
	scan string
	cell string
	keys []string
	cols []*sqlite3.Value
}

func declare(db *sqlite3.Conn, _, _, _ string, arg ...string) (_ *table, err error) {
	if len(arg) != 3 {
		return nil, fmt.Errorf("pivot: wrong number of arguments")
	}

	table := &table{db: db}
	defer func() {
		if err != nil {
			table.Close()
		}
	}()

	var sep string
	var create strings.Builder
	create.WriteString("CREATE TABLE x(")

	// Row key query.
	table.scan = "SELECT * FROM\n" + arg[0]
	stmt, _, err := db.Prepare(table.scan)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	table.keys = make([]string, stmt.ColumnCount())
	for i := range table.keys {
		name := sqlite3.QuoteIdentifier(stmt.ColumnName(i))
		table.keys[i] = name
		create.WriteString(sep)
		create.WriteString(name)
		sep = ","
	}
	stmt.Close()

	// Column definition query.
	stmt, _, err = db.Prepare("SELECT * FROM\n" + arg[1])
	if err != nil {
		return nil, err
	}

	if stmt.ColumnCount() != 2 {
		return nil, fmt.Errorf("pivot: column definition query expects 2 result columns")
	}
	for stmt.Step() {
		name := sqlite3.QuoteIdentifier(stmt.ColumnText(1))
		table.cols = append(table.cols, stmt.ColumnValue(0).Dup())
		create.WriteString(",")
		create.WriteString(name)
	}
	stmt.Close()

	// Pivot cell query.
	table.cell = "SELECT * FROM\n" + arg[2]
	stmt, _, err = db.Prepare(table.cell)
	if err != nil {
		return nil, err
	}

	if stmt.ColumnCount() != 1 {
		return nil, fmt.Errorf("pivot: cell query expects 1 result columns")
	}
	if stmt.BindCount() != len(table.keys)+1 {
		return nil, fmt.Errorf("pivot: cell query expects %d bound parameters", len(table.keys)+1)
	}

	create.WriteByte(')')
	err = db.DeclareVTab(create.String())
	if err != nil {
		return nil, err
	}
	return table, nil
}

func (t *table) Close() error {
	for i := range t.cols {
		t.cols[i].Close()
	}
	return nil
}

func (t *table) BestIndex(idx *sqlite3.IndexInfo) error {
	var idxStr strings.Builder
	idxStr.WriteString(t.scan)

	argvIndex := 1
	sep := " WHERE "
	for i, cst := range idx.Constraint {
		if !cst.Usable || !(0 <= cst.Column && cst.Column < len(t.keys)) {
			continue
		}
		op := operator(cst.Op)
		if op == "" {
			continue
		}

		idxStr.WriteString(sep)
		idxStr.WriteString(t.keys[cst.Column])
		idxStr.WriteString(" ")
		idxStr.WriteString(op)
		idxStr.WriteString(" ?")

		idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
			ArgvIndex: argvIndex,
			Omit:      true,
		}
		sep = " AND "
		argvIndex++
	}

	sep = " ORDER BY "
	idx.OrderByConsumed = true
	for _, ord := range idx.OrderBy {
		if !(0 <= ord.Column && ord.Column < len(t.keys)) {
			idx.OrderByConsumed = false
			continue
		}
		idxStr.WriteString(sep)
		idxStr.WriteString(t.keys[ord.Column])
		idxStr.WriteString(" COLLATE ")
		idxStr.WriteString(idx.Collation(ord.Column))
		if ord.Desc {
			idxStr.WriteString(" DESC")
		}
		sep = ","
	}

	idx.EstimatedCost = 1e9 / float64(argvIndex)
	idx.IdxStr = idxStr.String()
	return nil
}

func (t *table) Open() (sqlite3.VTabCursor, error) {
	return &cursor{table: t}, nil
}

func (t *table) Rename(new string) error {
	return nil
}

type cursor struct {
	table *table
	scan  *sqlite3.Stmt
	cell  *sqlite3.Stmt
	rowID int64
}

func (c *cursor) Close() error {
	return errors.Join(c.scan.Close(), c.cell.Close())
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	err := c.scan.Close()
	if err != nil {
		return err
	}

	c.scan, _, err = c.table.db.Prepare(idxStr)
	if err != nil {
		return err
	}
	for i, arg := range arg {
		err := c.scan.BindValue(i+1, arg)
		if err != nil {
			return err
		}
	}

	if c.cell == nil {
		c.cell, _, err = c.table.db.Prepare(c.table.cell)
		if err != nil {
			return err
		}
	}

	c.rowID = 0
	return c.Next()
}

func (c *cursor) Next() error {
	if c.scan.Step() {
		count := c.scan.ColumnCount()
		for i := 0; i < count; i++ {
			err := c.cell.BindValue(i+1, c.scan.ColumnValue(i))
			if err != nil {
				return err
			}
		}
		c.rowID++
	}
	return c.scan.Err()
}

func (c *cursor) EOF() bool {
	return !c.scan.Busy()
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx *sqlite3.Context, col int) error {
	count := c.scan.ColumnCount()
	if col < count {
		ctx.ResultValue(c.scan.ColumnValue(col))
		return nil
	}

	err := c.cell.BindValue(count+1, *c.table.cols[col-count])
	if err != nil {
		return err
	}

	if c.cell.Step() {
		ctx.ResultValue(c.cell.ColumnValue(0))
	}
	return c.cell.Reset()
}

func operator(op sqlite3.IndexConstraintOp) string {
	switch op {
	case sqlite3.INDEX_CONSTRAINT_EQ:
		return "="
	case sqlite3.INDEX_CONSTRAINT_LT:
		return "<"
	case sqlite3.INDEX_CONSTRAINT_GT:
		return ">"
	case sqlite3.INDEX_CONSTRAINT_LE:
		return "<="
	case sqlite3.INDEX_CONSTRAINT_GE:
		return ">="
	case sqlite3.INDEX_CONSTRAINT_NE:
		return "<>"
	case sqlite3.INDEX_CONSTRAINT_MATCH:
		return "MATCH"
	case sqlite3.INDEX_CONSTRAINT_LIKE:
		return "LIKE"
	case sqlite3.INDEX_CONSTRAINT_GLOB:
		return "GLOB"
	case sqlite3.INDEX_CONSTRAINT_REGEXP:
		return "REGEXP"
	case sqlite3.INDEX_CONSTRAINT_IS, sqlite3.INDEX_CONSTRAINT_ISNULL:
		return "IS"
	case sqlite3.INDEX_CONSTRAINT_ISNOT, sqlite3.INDEX_CONSTRAINT_ISNOTNULL:
		return "IS NOT"
	default:
		return ""
	}
}
