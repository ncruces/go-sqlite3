package tests

import (
	"os"
	"path/filepath"
	"testing"

	_ "embed"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

//go:embed testdata/wal.db
var walDB []byte

//go:embed testdata/utf16be.db
var utf16DB []byte

func TestDB_memory(t *testing.T) {
	t.Parallel()
	testDB(t, ":memory:")
}

func TestDB_file(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	t.Parallel()
	testDB(t, filepath.Join(t.TempDir(), "test.db"))
}

func TestDB_wal(t *testing.T) {
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	t.Parallel()
	tmp := filepath.Join(t.TempDir(), "test.db")
	err := os.WriteFile(tmp, walDB, 0666)
	if err != nil {
		t.Fatal(err)
	}
	testDB(t, tmp)
}

func TestDB_utf16(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	t.Parallel()
	tmp := filepath.Join(t.TempDir(), "test.db")
	err := os.WriteFile(tmp, utf16DB, 0666)
	if err != nil {
		t.Fatal(err)
	}
	testDB(t, tmp)
}

func TestDB_memdb(t *testing.T) {
	t.Parallel()
	testDB(t, memdb.TestDB(t))
}

func TestDB_adiantum(t *testing.T) {
	t.Parallel()
	tmp := filepath.Join(t.TempDir(), "test.db")
	testDB(t, "file:"+filepath.ToSlash(tmp)+"?nolock=1"+
		"&vfs=adiantum&textkey=correct+horse+battery+staple")
}

func TestDB_nolock(t *testing.T) {
	t.Parallel()
	tmp := filepath.Join(t.TempDir(), "test.db")
	testDB(t, "file:"+filepath.ToSlash(tmp)+"?nolock=1")
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

		if row >= 3 {
			continue
		}
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
