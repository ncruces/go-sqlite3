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

	err = sqlite3.CreateModule(db, "generate_series", seriesModule{})
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

type seriesModule struct{}

func (seriesModule) Connect(c *sqlite3.Conn, arg ...string) (*seriesTable, error) {
	err := c.DeclareVtab(`CREATE TABLE x(value, start HIDDEN, stop HIDDEN, step HIDDEN)`)
	if err != nil {
		return nil, err
	}
	return &seriesTable{0, 0, 1}, nil
}

type seriesTable struct {
	start int64
	stop  int64
	step  int64
}

func (*seriesTable) Disconnect() error {
	return nil
}

func (*seriesTable) BestIndex(idx *sqlite3.IndexInfo) error {
	idx.IdxNum = 0
	idx.IdxStr = "default"
	argv := 1
	for i, cst := range idx.Constraint {
		if cst.Op == sqlite3.Eq {
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				ArgvIndex: argv,
				Omit:      true,
			}
			argv++
		}
	}
	return nil
}

func (tab *seriesTable) Open() (sqlite3.VTabCursor, error) {
	return &seriesCursor{tab, 0}, nil
}

type seriesCursor struct {
	*seriesTable
	value int64
}

func (*seriesCursor) Close() error {
	return nil
}

func (cur *seriesCursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	switch len(arg) {
	case 0:
		cur.seriesTable.start = 0
		cur.seriesTable.stop = 1000
	case 1:
		cur.seriesTable.start = arg[0].Int64()
		cur.seriesTable.stop = 1000
	case 2:
		cur.seriesTable.start = arg[0].Int64()
		cur.seriesTable.stop = arg[1].Int64()
	case 3:
		cur.seriesTable.start = arg[0].Int64()
		cur.seriesTable.stop = arg[1].Int64()
		cur.seriesTable.step = arg[2].Int64()
	}
	cur.value = cur.seriesTable.start
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
