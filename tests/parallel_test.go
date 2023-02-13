package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestParallel(t *testing.T) {
	testParallel(t, t.TempDir(), 50)
}

func TestMultiProcess(t *testing.T) {
	if testing.Short() {
		return
	}

	dir := t.TempDir()
	t.Setenv("TestParallel_dir", dir)
	cmd := exec.Command("go", "test", "-run", "TestChildProcess")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	testParallel(t, dir, 500)
	cmd.Wait()
}

func TestChildProcess(t *testing.T) {
	dir := os.Getenv("TestParallel_dir")
	if dir == "" || testing.Short() {
		return
	}

	testParallel(t, dir, 500)
}

func testParallel(t *testing.T, dir string, n int) {
	writer := func() error {
		db, err := sqlite3.Open(filepath.Join(dir, "test.db"))
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.Exec(`
			PRAGMA locking_mode = NORMAL;
			PRAGMA busy_timeout = 10000;
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
			PRAGMA busy_timeout = 10000;
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

	err := writer()
	if err != nil {
		t.Fatal(err)
	}

	var group errgroup.Group
	group.SetLimit(4)
	for i := 0; i < n; i++ {
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
