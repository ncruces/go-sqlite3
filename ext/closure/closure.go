// Package closure provides a transitive closure virtual table.
//
// The transitive_closure virtual table finds the transitive closure of
// a parent/child relationship in a real table.
//
// https://sqlite.org/src/doc/tip/ext/misc/closure.c
package closure

import (
	"fmt"
	"math"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/vtabutil"
)

const (
	_COL_ID           = 0
	_COL_DEPTH        = 1
	_COL_ROOT         = 2
	_COL_TABLENAME    = 3
	_COL_IDCOLUMN     = 4
	_COL_PARENTCOLUMN = 5
)

// Register registers the transitive_closure virtual table:
//
//	CREATE VIRTUAL TABLE temp.closure USING transitive_closure;
func Register(db *sqlite3.Conn) error {
	return sqlite3.CreateModule(db, "transitive_closure", nil,
		func(db *sqlite3.Conn, _, _, _ string, arg ...string) (*closure, error) {
			var (
				table  string
				column string
				parent string

				done = util.Set[string]{}
			)

			for _, arg := range arg {
				key, val := vtabutil.NamedArg(arg)
				if done.Contains(key) {
					return nil, fmt.Errorf("transitive_closure: more than one %q parameter", key)
				}
				switch key {
				case "tablename":
					table = vtabutil.Unquote(val)
				case "idcolumn":
					column = vtabutil.Unquote(val)
				case "parentcolumn":
					parent = vtabutil.Unquote(val)
				default:
					return nil, fmt.Errorf("transitive_closure: unknown %q parameter", key)
				}
				done.Add(key)
			}

			err := db.DeclareVTab(`CREATE TABLE x(id,depth,root HIDDEN,tablename HIDDEN,idcolumn HIDDEN,parentcolumn HIDDEN)`)
			if err != nil {
				return nil, err
			}
			return &closure{
				db:     db,
				table:  table,
				column: column,
				parent: parent,
			}, nil
		})
}

type closure struct {
	db     *sqlite3.Conn
	table  string
	column string
	parent string
}

func (c *closure) Destroy() error { return nil }

func (c *closure) BestIndex(idx *sqlite3.IndexInfo) error {
	plan := 0
	posi := 1
	cost := 1e7

	for i, cst := range idx.Constraint {
		if !cst.Usable {
			continue
		}
		if plan&1 == 0 && cst.Column == _COL_ROOT {
			switch cst.Op {
			case sqlite3.INDEX_CONSTRAINT_EQ:
				plan |= 1
				cost /= 100
				idx.ConstraintUsage[i].ArgvIndex = 1
				idx.ConstraintUsage[i].Omit = true
			}
			continue
		}
		if plan&0xf0 == 0 && cst.Column == _COL_DEPTH {
			switch cst.Op {
			case sqlite3.INDEX_CONSTRAINT_LT, sqlite3.INDEX_CONSTRAINT_LE, sqlite3.INDEX_CONSTRAINT_EQ:
				plan |= posi << 4
				cost /= 5
				posi += 1
				idx.ConstraintUsage[i].ArgvIndex = posi
				if cst.Op == sqlite3.INDEX_CONSTRAINT_LT {
					plan |= 2
				}
			}
			continue
		}
		if plan&0xf00 == 0 && cst.Column == _COL_TABLENAME {
			switch cst.Op {
			case sqlite3.INDEX_CONSTRAINT_EQ:
				plan |= posi << 8
				cost /= 5
				posi += 1
				idx.ConstraintUsage[i].ArgvIndex = posi
				idx.ConstraintUsage[i].Omit = true
			}
			continue
		}
		if plan&0xf000 == 0 && cst.Column == _COL_IDCOLUMN {
			switch cst.Op {
			case sqlite3.INDEX_CONSTRAINT_EQ:
				plan |= posi << 12
				posi += 1
				idx.ConstraintUsage[i].ArgvIndex = posi
				idx.ConstraintUsage[i].Omit = true
			}
			continue
		}
		if plan&0xf0000 == 0 && cst.Column == _COL_PARENTCOLUMN {
			switch cst.Op {
			case sqlite3.INDEX_CONSTRAINT_EQ:
				plan |= posi << 16
				posi += 1
				idx.ConstraintUsage[i].ArgvIndex = posi
				idx.ConstraintUsage[i].Omit = true
			}
			continue
		}
	}

	if plan&1 == 0 ||
		c.table == "" && plan&0xf00 == 0 ||
		c.column == "" && plan&0xf000 == 0 ||
		c.parent == "" && plan&0xf0000 == 0 {
		return sqlite3.CONSTRAINT
	}

	idx.EstimatedCost = cost
	idx.IdxNum = plan
	return nil
}

func (c *closure) Open() (sqlite3.VTabCursor, error) {
	return &cursor{closure: c}, nil
}

type cursor struct {
	*closure
	nodes []node
}

type node struct {
	id    int64
	depth int
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	root := arg[0].Int64()
	maxDepth := math.MaxInt
	if idxNum&0xf0 != 0 {
		maxDepth = arg[(idxNum>>4)&0xf].Int()
		if idxNum&2 != 0 {
			maxDepth -= 1
		}
	}
	table := c.table
	if idxNum&0xf00 != 0 {
		table = arg[(idxNum>>8)&0xf].Text()
	}
	column := c.column
	if idxNum&0xf000 != 0 {
		column = arg[(idxNum>>12)&0xf].Text()
	}
	parent := c.parent
	if idxNum&0xf0000 != 0 {
		parent = arg[(idxNum>>16)&0xf].Text()
	}

	sql := fmt.Sprintf(
		`SELECT %[1]s.%[2]s FROM %[1]s WHERE %[1]s.%[3]s=?`,
		sqlite3.QuoteIdentifier(table),
		sqlite3.QuoteIdentifier(column),
		sqlite3.QuoteIdentifier(parent),
	)
	stmt, _, err := c.db.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	c.nodes = []node{{root, 0}}
	set := util.Set[int64]{}
	set.Add(root)
	for i := 0; i < len(c.nodes); i++ {
		curr := c.nodes[i]
		if curr.depth >= maxDepth {
			continue
		}
		stmt.BindInt64(1, curr.id)
		for stmt.Step() {
			if stmt.ColumnType(0) == sqlite3.INTEGER {
				next := stmt.ColumnInt64(0)
				if !set.Contains(next) {
					set.Add(next)
					c.nodes = append(c.nodes, node{next, curr.depth + 1})
				}
			}
		}
		stmt.Reset()
	}
	return nil
}

func (c *cursor) Column(ctx sqlite3.Context, n int) error {
	switch n {
	case _COL_ID:
		ctx.ResultInt64(c.nodes[0].id)
	case _COL_DEPTH:
		ctx.ResultInt(c.nodes[0].depth)
	case _COL_TABLENAME:
		ctx.ResultText(c.table)
	case _COL_IDCOLUMN:
		ctx.ResultText(c.column)
	case _COL_PARENTCOLUMN:
		ctx.ResultText(c.parent)
	}
	return nil
}

func (c *cursor) Next() error {
	c.nodes = c.nodes[1:]
	return nil
}

func (c *cursor) EOF() bool {
	return len(c.nodes) == 0
}

func (c *cursor) RowID() (int64, error) {
	return c.nodes[0].id, nil
}
