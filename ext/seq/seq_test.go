package seq

import (
	"fmt"
	"iter"
	"log"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func ExampleAggregate() {
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

	err = db.CreateWindowFunction("seq_avg", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS, Aggregate(
		func(seq iter.Seq[[]sqlite3.Value]) any {
			count := 0
			total := 0.0
			for arg := range seq {
				total += arg[0].Float()
				count++
			}
			return total / float64(count)
		}))
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
