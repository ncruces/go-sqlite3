package statement_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/statement"
	"github.com/tetratelabs/wazero"
)

func Example() {
	// This crashes the compiler.
	sqlite3.RuntimeConfig = wazero.NewRuntimeConfigInterpreter()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	statement.Register(db)

	err = db.Exec(`
		CREATE VIRTUAL TABLE split_date USING statement((
			SELECT
				strftime('%Y', :date) AS year,
				strftime('%m', :date) AS month,
				strftime('%d', :date) AS day
		))`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT * FROM split_date('2022-02-22')`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		fmt.Printf("Twosday was %d-%d-%d", stmt.ColumnInt(0), stmt.ColumnInt(1), stmt.ColumnInt(2))
	}
	if err := stmt.Reset(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// Twosday was 2022-2-22
}
