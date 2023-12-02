package blob_test

import (
	"io"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/array"
	"github.com/ncruces/go-sqlite3/ext/blob"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example() {
	// Open the database, registering the extension.
	db, err := driver.Open("file:/test.db?vfs=memdb", func(conn *sqlite3.Conn) error {
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

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	blob.Register(db)
	array.Register(db)

	err = db.Exec(`SELECT blob_open()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test1 (col);
		CREATE TABLE IF NOT EXISTS test2 (col);
		INSERT INTO test1 VALUES (x'cafe');
		INSERT INTO test2 VALUES (x'babe');
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT blob_open('main', value, 'col', 1, false, ?) FROM array(?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	var got []string
	err = stmt.BindPointer(1, blob.OpenCallback(func(b *sqlite3.Blob, _ ...sqlite3.Value) error {
		d, err := io.ReadAll(b)
		if err != nil {
			return err
		}
		got = append(got, string(d))
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.BindPointer(2, []string{"test1", "test2"})
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"\xca\xfe", "\xba\xbe"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
