package sqlite3_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestOpen_memory(t *testing.T) {
	testOpen(t, ":memory:")
}

func TestOpen_file(t *testing.T) {
	dir, err := os.MkdirTemp("", "sqlite3-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	testOpen(t, filepath.Join(dir, "test.db"))
}

func TestOpen_dir(t *testing.T) {
	_, err := sqlite3.Open(".")
	if err == nil {
		t.Fatal("want error")
	}
	var serr *sqlite3.Error
	if !errors.As(err, &serr) {
		t.Fatal("want sqlite3.Error")
	}
	if serr.Code != sqlite3.CANTOPEN {
		t.Error("want sqlite3.CANTOPEN")
	}
	if got := err.Error(); got != "sqlite3: unable to open database file" {
		t.Error("got message: ", got)
	}
}

func testOpen(t *testing.T, name string) {
	db, err := sqlite3.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO users(id, name) VALUES(0, 'go'), (1, 'zig'), (2, 'whatever')`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
	if err != nil {
		t.Fatal(err)
	}

	ids := []int{0, 1, 2}
	names := []string{"go", "zig", "whatever"}

	idx := 0
	for ; stmt.Step(); idx++ {
		if ids[idx] != stmt.ColumnInt(0) {
			t.Errorf("got %d, want %d", stmt.ColumnInt(0), ids[idx])
		}
		if names[idx] != stmt.ColumnText(1) {
			t.Errorf("got %q, want %q", stmt.ColumnText(1), names[idx])
		}
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}
	if idx != 3 {
		t.Errorf("got %d rows, want %d", idx, len(ids))
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}
