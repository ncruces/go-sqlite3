package memdb_test

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

//go:embed testdata/test.db
var testDB []byte

func Example() {
	memdb.Create("test.db", testDB)

	db, err := sql.Open("sqlite3", "file:/test.db?vfs=memdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO users (id, name) VALUES (3, 'rust')`)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query(`SELECT id, name FROM users`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s\n", id, name)
	}
	// Output:
	// 0 go
	// 1 zig
	// 2 whatever
	// 3 rust
}
