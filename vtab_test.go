package sqlite3_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func ExampleCreateModule() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = sqlite3.CreateModule[seriesTable](db, "generate_series", nil,
		func(db *sqlite3.Conn, module, schema, table string, arg ...string) (seriesTable, error) {
			err := db.DeclareVTab(`CREATE TABLE x(value, start HIDDEN, stop HIDDEN, step HIDDEN)`)
			return seriesTable{}, err
		})
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT rowid, value FROM generate_series(2, 10, 3)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnInt(0), stmt.ColumnInt(1))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// 2 2
	// 5 5
	// 8 8
}

type seriesTable struct{}

func (seriesTable) BestIndex(idx *sqlite3.IndexInfo) error {
	for i, cst := range idx.Constraint {
		switch cst.Column {
		case 1, 2, 3: // start, stop, step
			if cst.Op == sqlite3.INDEX_CONSTRAINT_EQ && cst.Usable {
				idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
					ArgvIndex: cst.Column,
					Omit:      true,
				}
			}
		}
	}
	idx.IdxNum = 1
	idx.IdxStr = "idx"
	return nil
}

func (seriesTable) Open() (sqlite3.VTabCursor, error) {
	return &seriesCursor{}, nil
}

type seriesCursor struct {
	start int64
	stop  int64
	step  int64
	value int64
}

func (cur *seriesCursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	if idxNum != 1 || idxStr != "idx" {
		return nil
	}
	cur.start = 0
	cur.stop = 1000
	cur.step = 1
	if len(arg) > 0 {
		cur.start = arg[0].Int64()
	}
	if len(arg) > 1 {
		cur.stop = arg[1].Int64()
	}
	if len(arg) > 2 {
		cur.step = arg[2].Int64()
	}
	cur.value = cur.start
	return nil
}

func (cur *seriesCursor) Column(ctx *sqlite3.Context, col int) error {
	switch col {
	case 0:
		ctx.ResultInt64(cur.value)
	case 1:
		ctx.ResultInt64(cur.start)
	case 2:
		ctx.ResultInt64(cur.stop)
	case 3:
		ctx.ResultInt64(cur.step)
	}
	return nil
}

func (cur *seriesCursor) Next() error {
	cur.value += cur.step
	return nil
}

func (cur *seriesCursor) EOF() bool {
	return cur.value > cur.stop
}

func (cur *seriesCursor) RowID() (int64, error) {
	return int64(cur.value), nil
}
