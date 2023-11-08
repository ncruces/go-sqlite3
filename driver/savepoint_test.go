package driver_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func ExampleSavepoint() {
	db, err := driver.Open("file:/test.db?vfs=memdb", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	err = func() error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare(`INSERT INTO users (id, name) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		_, err = stmt.Exec(0, "go")
		if err != nil {
			return err
		}

		_, err = stmt.Exec(1, "zig")
		if err != nil {
			return err
		}

		savept := driver.Savepoint(tx)

		_, err = stmt.Exec(2, "whatever")
		if err != nil {
			return err
		}

		err = savept.Rollback()
		if err != nil {
			return err
		}

		_, err = stmt.Exec(3, "rust")
		if err != nil {
			return err
		}

		return tx.Commit()
	}()
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
	// 3 rust
}
