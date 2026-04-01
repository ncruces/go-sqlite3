//go:build !js

package csv_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/csv"
)

func Example() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = csv.Register(db)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`
		CREATE VIRTUAL TABLE eurofxref USING csv(
			filename = 'testdata/eurofxref.csv',
			header   = YES,
			columns  = 42,
		)`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT USD FROM eurofxref WHERE Date = '2022-02-22'`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		fmt.Printf("On Twosday, 1€ = $%g", stmt.ColumnFloat(0))
	}
	if err := stmt.Reset(); err != nil {
		log.Fatal(err)
	}

	err = db.Exec(`DROP TABLE eurofxref`)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// On Twosday, 1€ = $1.1342
}
