//go:build !sqlite3_nosys

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/tests/testcfg"
	"github.com/ncruces/go-sqlite3/vfs"
)

func TestWAL_enter_exit(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "test.db")

	db, err := sqlite3.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE test (col);
		PRAGMA journal_mode=WAL;
		SELECT * FROM test;
		PRAGMA journal_mode=DELETE;
		SELECT * FROM test;
		PRAGMA journal_mode=WAL;
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

	tmp := filepath.Join(t.TempDir(), "test.db")
	err := os.WriteFile(tmp, waldb, 0666)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sqlite3.OpenFlags(tmp, sqlite3.OPEN_READONLY)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT * FROM sqlite_master`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		t.Error("want no rows")
	}
}

func TestConn_WalCheckpoint(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "test.db")

	db, err := sqlite3.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.WalAutoCheckpoint(1000)
	if err != nil {
		t.Fatal(err)
	}

	db.WalHook(func(db *sqlite3.Conn, schema string, pages int) error {
		log, ckpt, err := db.WalCheckpoint(schema, sqlite3.CHECKPOINT_FULL)
		t.Log(log, ckpt, err)
		return err
	})

	err = db.Exec(`
		PRAGMA journal_mode=WAL;
		CREATE TABLE test (col);
	`)
	if err != nil {
		t.Fatal(err)
	}
}
