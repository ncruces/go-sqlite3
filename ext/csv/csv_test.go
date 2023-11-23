package csv_test

import (
	"fmt"
	"log"
	"os"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/csv"
)

func Example() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	csv.Register(db, os.Open)

	err = db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS eurofxref USING csv(
			filename = 'eurofxref.csv',
			header   = YES,
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
