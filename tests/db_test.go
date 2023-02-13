package tests

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestDB_memory(t *testing.T) {
	testDB(t, ":memory:")
}

func TestDB_file(t *testing.T) {
	testDB(t, filepath.Join(t.TempDir(), "test.db"))
}

func testDB(t *testing.T, name string) {
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
	defer stmt.Close()

	row := 0
	ids := []int{0, 1, 2}
	names := []string{"go", "zig", "whatever"}
	for ; stmt.Step(); row++ {
		if ids[row] != stmt.ColumnInt(0) {
			t.Errorf("got %d, want %d", stmt.ColumnInt(0), ids[row])
		}
		if names[row] != stmt.ColumnText(1) {
			t.Errorf("got %q, want %q", stmt.ColumnText(1), names[row])
		}
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}
	if row != 3 {
		t.Errorf("got %d rows, want %d", row, len(ids))
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
