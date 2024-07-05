package blobio_test

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
	"github.com/ncruces/go-sqlite3/ext/blobio"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example() {
	// Open the database, registering the extension.
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

	// Create the BLOB.
	_, err = db.Exec(`INSERT INTO test VALUES (?)`, sqlite3.ZeroBlob(len(message)))
	if err != nil {
		log.Fatal(err)
	}

	// Write the BLOB.
	_, err = db.Exec(`SELECT writeblob('main', 'test', 'col', last_insert_rowid(), 0, ?)`, message)
	if err != nil {
		log.Fatal(err)
	}

	// Read the BLOB.
	_, err = db.Exec(`SELECT openblob('main', 'test', 'col', rowid, false, ?) FROM test`,
		sqlite3.Pointer[blobio.OpenCallback](func(blob *sqlite3.Blob, _ ...sqlite3.Value) error {
			_, err = io.Copy(os.Stdout, blob)
			return err
		}))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello BLOB!
}

func Test_readblob(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	blobio.Register(db)
	array.Register(db)

	err = db.Exec(`SELECT readblob()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`
		CREATE TABLE test1 (col);
		CREATE TABLE test2 (col);
		INSERT INTO test1 VALUES (x'cafe');
		INSERT INTO test2 VALUES (x'babe');
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT readblob('main', value, 'col', 1, 1, 1) FROM array(?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.BindPointer(1, []string{"test1", "test2"})
	if err != nil {
		t.Fatal(err)
	}

	if stmt.Step() {
		got := stmt.ColumnText(0)
		if got != "\xfe" {
			t.Errorf("got %q", got)
		}
	}

	if stmt.Step() {
		got := stmt.ColumnText(0)
		if got != "\xbe" {
			t.Errorf("got %q", got)
		}
	}

	err = stmt.Err()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_openblob(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	blobio.Register(db)
	array.Register(db)

	err = db.Exec(`SELECT openblob()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`
		CREATE TABLE test1 (col);
		CREATE TABLE test2 (col);
		INSERT INTO test1 VALUES (x'cafe');
		INSERT INTO test2 VALUES (x'babe');
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT openblob('main', value, 'col', 1, false, ?) FROM array(?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	var got []string
	err = stmt.BindPointer(1, blobio.OpenCallback(func(b *sqlite3.Blob, _ ...sqlite3.Value) error {
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
