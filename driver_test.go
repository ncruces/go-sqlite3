package sqlite3_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const demo = "demo.db"

func ExampleDriverConn() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sql.Open("sqlite3", demo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		log.Fatal(err)
	}

	r, err := conn.ExecContext(ctx, `INSERT INTO test VALUES (?)`, sqlite3.ZeroBlob(11))
	if err != nil {
		log.Fatal(err)
	}

	id, err := r.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Raw(func(driverConn any) error {
		conn := driverConn.(sqlite3.DriverConn)
		defer conn.Savepoint()(&err)

		blob, err := conn.OpenBlob("main", "test", "col", id, true)
		if err != nil {
			return err
		}
		defer blob.Close()

		_, err = fmt.Fprint(blob, "Hello BLOB!")
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	var msg string
	err = conn.QueryRowContext(ctx, `SELECT col FROM test`).Scan(&msg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg)
	// Output:
	// Hello BLOB!
}
