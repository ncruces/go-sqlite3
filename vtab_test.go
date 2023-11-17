package sqlite3_test

import (
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

	stmt, _, err := db.Prepare(`SELECT value FROM generate_series(5,100,5)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	// Output:
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
		if cst.Usable && cst.Op == sqlite3.Eq {
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				ArgvIndex: argv,
				Omit:      true,
			}
			argv++
		}
	}
	return nil
}

func (*seriesTable) Open() (sqlite3.VTabCursor, error) { return nil, nil }
