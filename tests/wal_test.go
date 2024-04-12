package tests

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
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
