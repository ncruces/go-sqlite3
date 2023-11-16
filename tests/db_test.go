package tests

import (
	"os"
	"path/filepath"
	"testing"

	_ "embed"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

//go:embed testdata/wal.db
var waldb []byte

func TestDB_memory(t *testing.T) {
	t.Parallel()
	testDB(t, ":memory:")
}

func TestDB_file(t *testing.T) {
	t.Parallel()
	testDB(t, filepath.Join(t.TempDir(), "test.db"))
}

func TestDB_nolock(t *testing.T) {
	t.Parallel()
	testDB(t, "file:"+
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))+
		"?nolock=1")
}

func TestDB_wal(t *testing.T) {
	t.Parallel()
	wal := filepath.Join(t.TempDir(), "test.db")
	err := os.WriteFile(wal, waldb, 0666)
	if err != nil {
		t.Fatal(err)
	}
	testDB(t, wal)
}

func TestDB_vfs(t *testing.T) {
	testDB(t, "file:test.db?vfs=memdb")
}

func testDB(t testing.TB, name string) {
	db, err := sqlite3.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
	if err != nil {
		t.Fatal(err)
	}
	changes := db.Changes()
	if changes != 3 {
		t.Errorf("got %d want 3", changes)
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
		id := stmt.ColumnInt(0)
		name := stmt.ColumnText(1)

		if id != ids[row] {
			t.Errorf("got %d, want %d", id, ids[row])
		}
		if name != names[row] {
			t.Errorf("got %q, want %q", name, names[row])
		}
	}
	if row != 3 {
		t.Errorf("got %d, want %d", row, len(ids))
	}

	if err := stmt.Err(); err != nil {
		t.Fatal(err)
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
