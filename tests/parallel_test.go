package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestParallel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	dir, err := os.MkdirTemp("", "sqlite3-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writer := func() error {
		db, err := sqlite3.Open(filepath.Join(dir, "test.db"))
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.Exec(`
			PRAGMA locking_mode = NORMAL;
			PRAGMA busy_timeout = 1000;
		`)
		if err != nil {
			return err
		}

		err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
		if err != nil {
			return err
		}

		err = db.Exec(`INSERT INTO users(id, name) VALUES(0, 'go'), (1, 'zig'), (2, 'whatever')`)
		if err != nil {
			return err
		}

		return db.Close()
	}

	reader := func() error {
		db, err := sqlite3.Open(filepath.Join(dir, "test.db"))
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.Exec(`
			PRAGMA locking_mode = NORMAL;
			PRAGMA busy_timeout = 1000;
		`)
		if err != nil {
			return err
		}

		stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
		if err != nil {
			return err
		}

		row := 0
		for stmt.Step() {
			row++
		}
		if err := stmt.Err(); err != nil {
			return err
		}
		if row%3 != 0 {
			t.Errorf("got %d rows, want multiple of 3", row)
		}

		err = stmt.Close()
		if err != nil {
			return err
		}

		return db.Close()
	}

	err = writer()
	if err != nil {
		t.Fatal(err)
	}

	var group errgroup.Group
	group.SetLimit(4)
	for i := 0; i < 32; i++ {
		if i&7 != 7 {
			group.Go(reader)
		} else {
			group.Go(writer)
		}
	}
	err = group.Wait()
	if err != nil {
		t.Fatal(err)
	}
}
