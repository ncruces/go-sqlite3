package blobio_test

import (
	"io"
	"log"
	"os"
	"slices"
	"strings"
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
	r, err := db.Exec(`INSERT INTO test VALUES (?)`, sqlite3.ZeroBlob(len(message)))
	if err != nil {
		log.Fatal(err)
	}

	id, err := r.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	// Write the BLOB.
	_, err = db.Exec(`SELECT writeblob('main', 'test', 'col', ?, 0, ?)`,
		id, message)
	if err != nil {
		log.Fatal(err)
	}

	// Read the BLOB.
	_, err = db.Exec(`SELECT readblob('main', 'test', 'col', ?, 0, ?)`,
		id, sqlite3.Pointer(os.Stdout))
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Hello BLOB!
}

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(blobio.Register)
	sqlite3.AutoExtension(array.Register)
	os.Exit(m.Run())
}

func Test_readblob(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`SELECT readblob()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`SELECT readblob('main', 'test1', 'col', 1, 1, 1)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`
		CREATE TABLE test1 (col);
		CREATE TABLE test2 (col);
		INSERT INTO test1 VALUES (x'cafe');
		INSERT INTO test1 VALUES (x'dead');
		INSERT INTO test2 VALUES (x'babe');
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`SELECT readblob('main', 'test1', 'col', 1, -1, 1)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`SELECT readblob('main', 'test1', 'col', 1, 1, 0)`)
	if err != nil {
		t.Log(err)
	}

	tests := []struct {
		name  string
		sql   string
		want1 string
		want2 string
	}{
		{"rows", `SELECT readblob('main', 'test1', 'col', rowid, 1, 1) FROM test1`, "\xfe", "\xad"},
		{"tables", `SELECT readblob('main', value, 'col', 1, 1, 1) FROM array(?)`, "\xfe", "\xbe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, _, err := db.Prepare(tt.sql)
			if err != nil {
				t.Fatal(err)
			}
			defer stmt.Close()

			if stmt.BindCount() == 1 {
				err = stmt.BindPointer(1, []string{"test1", "test2"})
				if err != nil {
					t.Fatal(err)
				}
			}

			if stmt.Step() {
				got := stmt.ColumnText(0)
				if got != tt.want1 {
					t.Errorf("got %q", got)
				}
			}

			if stmt.Step() {
				got := stmt.ColumnText(0)
				if got != tt.want2 {
					t.Errorf("got %q", got)
				}
			}

			err = stmt.Err()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_writeblob(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`SELECT writeblob()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`SELECT writeblob('main', 'test', 'col', 1, 1, x'')`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`
		CREATE TABLE test (col);
		INSERT INTO test VALUES (x'cafe');
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`SELECT writeblob('main', 'test', 'col', 1, -1, x'')`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	stmt, _, err := db.Prepare(`SELECT writeblob('main', 'test', 'col', 1, 0, ?)`)
	if err != nil {
		t.Log(err)
	}
	defer stmt.Close()

	err = stmt.BindPointer(1, strings.NewReader("\xba\xbe"))
	if err != nil {
		t.Log(err)
	}

	err = stmt.Exec()
	if err != nil {
		t.Log(err)
	}
}

func Test_openblob(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`SELECT openblob()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`SELECT openblob('main', 'test1', 'col', 1, false, NULL)`)
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
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
