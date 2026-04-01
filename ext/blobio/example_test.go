//go:build !js

package blobio_test

import (
	"database/sql"
	"log"
	"os"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/blobio"
)

func Example() {
	db, err := driver.Open("file:/test.db?vfs=memdb", blobio.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		log.Fatal(err)
	}

	const message = "Hello BLOB!"

	r, err := db.Exec(`INSERT INTO test VALUES (:data)`,
		sql.Named("data", sqlite3.ZeroBlob(len(message))))
	if err != nil {
		log.Fatal(err)
	}

	id, err := r.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`SELECT writeblob('main', 'test', 'col', :rowid, :offset, :message)`,
		sql.Named("rowid", id),
		sql.Named("offset", 0),
		sql.Named("message", message))
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`SELECT readblob('main', 'test', 'col', :rowid, :offset, :writer)`,
		sql.Named("rowid", id),
		sql.Named("offset", 0),
		sql.Named("writer", sqlite3.Pointer(os.Stdout)))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello BLOB!
}
