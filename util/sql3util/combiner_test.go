package sql3util_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func ExampleWindowFunction() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE flags (mask INTEGER)`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO flags VALUES (1), (2), (4), (4), (8)`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateWindowFunction("bit_or", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS,
		sql3util.WindowFunction(func() *bitOr { return &bitOr{} }))
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT bit_or(mask) OVER (ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM flags`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Printf("%04b\n", stmt.ColumnInt64(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// 0011
	// 0111
	// 0110
	// 1100
	// 1100
}

type bitOr struct {
	val int64
	ok  bool
}

func (b *bitOr) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.val |= arg[0].Int64()
		b.ok = true
	}
}

func (b *bitOr) Combine(other *bitOr) {
	if other.ok {
		b.val |= other.val
		b.ok = true
	}
}

func (b *bitOr) Value(ctx sqlite3.Context) {
	if b.ok {
		ctx.ResultInt64(b.val)
	}
}
