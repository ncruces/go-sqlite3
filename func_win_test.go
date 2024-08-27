package sqlite3_test

import (
	"fmt"
	"log"
	"unicode"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func ExampleConn_CreateWindowFunction() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE words (word VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateWindowFunction("count_ascii", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS, newASCIICounter)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT count_ascii(word) OVER (ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM words`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnInt(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// 1
	// 2
	// 2
	// 1
	// 0
	// 0
}

type countASCII struct{ result int }

func newASCIICounter() sqlite3.AggregateFunction {
	return &countASCII{}
}

func (f *countASCII) Value(ctx sqlite3.Context) {
	ctx.ResultInt(f.result)
}

func (f *countASCII) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if f.isASCII(arg[0]) {
		f.result++
	}
}

func (f *countASCII) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if f.isASCII(arg[0]) {
		f.result--
	}
}

func (f *countASCII) isASCII(arg sqlite3.Value) bool {
	if arg.Type() != sqlite3.TEXT {
		return false
	}
	for _, c := range arg.RawText() {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}
