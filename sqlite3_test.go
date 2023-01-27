package sqlite3_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestDB_memory(t *testing.T) {
	testDB(t, ":memory:")
}

func TestDB_file(t *testing.T) {
	dir, err := os.MkdirTemp("", "sqlite3-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	testDB(t, filepath.Join(dir, "test.db"))
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

func TestDB_parallel(t *testing.T) {
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
			t.Fatal(err)
		}

		err = db.Exec(`INSERT INTO users(id, name) VALUES(0, 'go'), (1, 'zig'), (2, 'whatever')`)
		if err != nil {
			t.Fatal(err)
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
		time.Sleep(time.Microsecond)
	}
	err = group.Wait()
	if err != nil {
		t.Fatal(err)
	}
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
