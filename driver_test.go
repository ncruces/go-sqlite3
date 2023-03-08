package sqlite3_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var db *sql.DB

func ExampleDriverConn() {
	var err error
	db, err = sql.Open("sqlite3", "demo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove("demo.db")
	defer db.Close()

	ctx := context.Background()

	conn, err := db.Conn(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		log.Fatal(err)
	}

	res, err := conn.ExecContext(ctx, `INSERT INTO test VALUES (?)`, sqlite3.ZeroBlob(11))
	if err != nil {
		log.Fatal(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Raw(func(driverConn any) error {
		conn := driverConn.(sqlite3.DriverConn)
		savept := conn.Savepoint()
		defer savept.Release(&err)

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
