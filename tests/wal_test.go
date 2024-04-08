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
