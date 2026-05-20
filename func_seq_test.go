package sqlite3_test

import (
	"fmt"
	"iter"
	"log"
	"testing"

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

// TestAggregateSeqFunction_EmptyInput verifies that the per-group
// callback is invoked exactly once even when the group has zero
// input rows, matching SQLite's xFinal semantics.
func TestAggregateSeqFunction_EmptyInput(t *testing.T) {
	db, err := sqlite3.Open(memory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var calls int
	err = db.CreateAggregateFunction("my_count", 0, 0,
		func(ctx *sqlite3.Context, seq iter.Seq[[]sqlite3.Value]) {
			calls++
			var n int64
			for range seq {
				n++
			}
			ctx.ResultInt64(n)
		})
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT my_count() FROM (SELECT 1) WHERE FALSE`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	if !stmt.Step() {
		t.Fatalf("no row: %v", stmt.Err())
	}
	if got := stmt.ColumnType(0); got == sqlite3.NULL {
		t.Errorf("aggregate over empty input: got NULL, want 0")
	}
	if got, want := stmt.ColumnInt64(0), int64(0); got != want {
		t.Errorf("my_count() over empty input = %d, want %d", got, want)
	}
	if calls != 1 {
		t.Errorf("callback invocations = %d, want 1", calls)
	}
}
