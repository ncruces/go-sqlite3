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

	err = db.Exec(`CREATE TABLE IF NOT EXISTS words (word VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		log.Fatal(err)
	}

	err = db.CreateWindowFunction("count_ascii", 1, sqlite3.INNOCUOUS, countASCII{})
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT count_ascii(word) FROM words`)
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

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 2
}

type countASCII struct{}

func (countASCII) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].Type() != sqlite3.TEXT {
		return
	}
	for _, c := range arg[0].RawText() {
		if c > unicode.MaxASCII {
			return
		}
	}
	if count := sqlite3.AggregateContext[int](ctx); count != nil {
		*count++
	}
}

func (countASCII) Final(ctx sqlite3.Context) {
	if count := sqlite3.AggregateContext[int](ctx); count != nil {
		ctx.ResultInt(*count)
	}
}
