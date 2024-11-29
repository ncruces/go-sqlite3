package sqlite3_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
)

func ExampleConn_CreateBulk() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE users (id INT, name VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	bulk, err := db.CreateBulk(`INSERT INTO users (id, name) VALUES`, ``)
	if err != nil {
		log.Fatal(err)
	}
	defer bulk.Close()

	for _, row := range [][]any{
		{0, "go"},
		{1, "zig"},
		{2, "whatever"},
	} {
		err = bulk.AppendRow(row...)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = bulk.Flush()
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnInt(0), stmt.ColumnText(1))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// 0 go
	// 1 zig
	// 2 whatever
}
