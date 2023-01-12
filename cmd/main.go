package main

import (
	"log"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func main() {
	db, err := sqlite3.Open(":memory:", sqlite3.SQLITE_OPEN_READWRITE|sqlite3.SQLITE_OPEN_CREATE, "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
