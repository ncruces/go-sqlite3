package fts5_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example() {
	db, err := driver.Open("file:/test.db?vfs=memdb", fts5.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE docs USING fts5(title, body);
		INSERT INTO docs(title, body) VALUES 
			('Go Programming', 'An intensive guide to Go routines.'),
			('SQLite Tutorial', 'Learn how to use virtual tables efficiently.');
	`)
	if err != nil {
		log.Fatal(err)
	}

	var title string
	err = db.QueryRow("SELECT title FROM docs WHERE docs MATCH 'Go AND routines'").Scan(&title)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(title)
	// Output: Go Programming
}
