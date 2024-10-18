package tests

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs"
)

func TestWAL_enter_exit(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}
	t.Parallel()

	file := filepath.Join(t.TempDir(), "test.db")

	db, err := sqlite3.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if !vfs.SupportsSharedMemory {
		err = db.Exec(`PRAGMA locking_mode=exclusive`)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = db.Exec(`
		CREATE TABLE test (col);
		PRAGMA journal_mode=wal;
		SELECT * FROM test;
		PRAGMA journal_mode=delete;
		SELECT * FROM test;
		PRAGMA journal_mode=wal;
		SELECT * FROM test;
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWAL_readonly(t *testing.T) {
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}
	t.Parallel()

	tmp := filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))

	db1, err := driver.Open("file:" + tmp + "?_pragma=journal_mode(wal)&_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()

	db2, err := driver.Open("file:" + tmp + "?_pragma=journal_mode(wal)&mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	// Create the table using the first (writable) connection.
	_, err = db1.Exec(`
		CREATE TABLE t(id INTEGER PRIMARY KEY, name TEXT);
		INSERT INTO t(name) VALUES('alice');
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Select the data using the second (readonly) connection.
	var name string
	err = db2.QueryRow("SELECT name FROM t").Scan(&name)
	if err != nil {
		t.Fatal(err)
	}
	if name != "alice" {
		t.Errorf("got %q want alice", name)
	}

	// Update table.
	_, err = db1.Exec(`
		DELETE FROM t;
		INSERT INTO t(name) VALUES('bob');
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Select the data using the second (readonly) connection.
	err = db2.QueryRow("SELECT name FROM t").Scan(&name)
	if err != nil {
		t.Fatal(err)
	}
	if name != "bob" {
		t.Errorf("got %q want bob", name)
	}
}

func TestConn_WALCheckpoint(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}
	t.Parallel()

	file := filepath.Join(t.TempDir(), "test.db")

	db, err := sqlite3.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.WALAutoCheckpoint(1000)
	if err != nil {
		t.Fatal(err)
	}

	db.WALHook(func(db *sqlite3.Conn, schema string, pages int) error {
		log, ckpt, err := db.WALCheckpoint(schema, sqlite3.CHECKPOINT_FULL)
		t.Log(log, ckpt, err)
		return err
	})

	err = db.Exec(`
		PRAGMA locking_mode=exlusive;
		PRAGMA journal_mode=wal;
		CREATE TABLE test (col);
	`)
	if err != nil {
		t.Fatal(err)
	}
}
