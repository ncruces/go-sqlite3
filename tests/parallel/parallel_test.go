package tests

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/tests/testcfg"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestParallel(t *testing.T) {
	var iter int
	if testing.Short() {
		iter = 1000
	} else {
		iter = 5000
	}

	name := "file:" +
		filepath.Join(t.TempDir(), "test.db") +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func TestWAL(t *testing.T) {
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	name := "file:" +
		filepath.Join(t.TempDir(), "test.db") +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(wal)" +
		"&_pragma=synchronous(off)"
	testParallel(t, name, 1000)
	testIntegrity(t, name)
}

func TestMemory(t *testing.T) {
	var iter int
	if testing.Short() {
		iter = 1000
	} else {
		iter = 5000
	}

	name := "file:/test.db?vfs=memdb"
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func TestMultiProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	file := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("TestMultiProcess_dbfile", file)

	name := "file:" + file +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"

	cmd := exec.Command(os.Args[0], append(os.Args[1:], "-test.v", "-test.run=TestChildProcess")...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	var buf [3]byte
	// Wait for child to start.
	if _, err := io.ReadFull(out, buf[:]); err != nil {
		t.Fatal(err)
	} else if str := string(buf[:]); str != "===" {
		t.Fatal(str)
	}

	testParallel(t, name, 1000)
	if err := cmd.Wait(); err != nil {
		t.Error(err)
	}
	testIntegrity(t, name)
}

func TestChildProcess(t *testing.T) {
	file := os.Getenv("TestMultiProcess_dbfile")
	if file == "" || testing.Short() {
		t.SkipNow()
	}

	name := "file:" + file +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"

	testParallel(t, name, 1000)
}

func BenchmarkMemory(b *testing.B) {
	memdb.Delete("test.db")
	name := "file:/test.db?vfs=memdb"
	testParallel(b, name, b.N)
}

func testParallel(t testing.TB, name string, n int) {
	writer := func() error {
		db, err := sqlite3.Open(name)
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.BusyHandler(func(count int) (retry bool) {
			time.Sleep(time.Millisecond)
			return true
		})
		if err != nil {
			return err
		}

		err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
		if err != nil {
			return err
		}

		err = db.Exec(`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
		if err != nil {
			return err
		}

		return db.Close()
	}

	reader := func() error {
		db, err := sqlite3.Open(name)
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.BusyTimeout(10 * time.Second)
		if err != nil {
			return err
		}

		stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
		if err != nil {
			return err
		}
		defer stmt.Close()

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
	group.SetLimit(6)
	for i := 0; i < n; i++ {
		if i&7 != 7 {
			group.Go(reader)
		} else {
			group.Go(writer)
		}
	}
	err = group.Wait()
	if err != nil {
		t.Error(err)
	}
}

func testIntegrity(t testing.TB, name string) {
	db, err := sqlite3.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	test := `PRAGMA integrity_check`
	if testing.Short() {
		test = `PRAGMA quick_check`
	}

	stmt, _, err := db.Prepare(test)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		if row := stmt.ColumnText(0); row != "ok" {
			t.Error(row)
		}
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
