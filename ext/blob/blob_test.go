package blob_test

import (
	"io"
	"log"
	"os"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/blob"
)

func Example() {
	// Open the database, registering the extension.
	db, err := driver.Open(":memory:", func(conn *sqlite3.Conn) error {
		blob.Register(conn)
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		log.Fatal(err)
	}

	const message = "Hello BLOB!"

	// Create the BLOB.
	_, err = db.Exec(`INSERT INTO test VALUES (?)`, sqlite3.ZeroBlob(len(message)))
	if err != nil {
		log.Fatal(err)
	}

	// Write the BLOB.
	_, err = db.Exec(`SELECT blob_open('main', 'test', 'col', last_insert_rowid(), true, ?)`,
		sqlite3.Pointer[blob.OpenCallback](func(blob *sqlite3.Blob, _ ...sqlite3.Value) error {
			_, err = io.WriteString(blob, message)
			return err
		}))
	if err != nil {
		log.Fatal(err)
	}

	// Read the BLOB.
	_, err = db.Exec(`SELECT blob_open('main', 'test', 'col', rowid, false, ?) FROM test`,
		sqlite3.Pointer[blob.OpenCallback](func(blob *sqlite3.Blob, _ ...sqlite3.Value) error {
			_, err = io.Copy(os.Stdout, blob)
			return err
		}))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello BLOB!
}
