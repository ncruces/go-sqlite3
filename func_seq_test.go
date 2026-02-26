package sqlite3_test

import (
	"fmt"
	"iter"
	"log"

	"github.com/ncruces/go-sqlite3"
)

func ExampleConn_CreateAggregateFunction() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (1), (2), (3)`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateAggregateFunction("seq_avg", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS,
		func(ctx *sqlite3.Context, seq iter.Seq[[]sqlite3.Value]) {
			count := 0
			total := 0.0
			for arg := range seq {
				switch arg[0].NumericType() {
				case sqlite3.FLOAT, sqlite3.INTEGER:
					total += arg[0].Float()
					count++
				}
			}
			ctx.ResultFloat(total / float64(count))
		})
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT seq_avg(col) FROM test`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnFloat(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// 2
}
