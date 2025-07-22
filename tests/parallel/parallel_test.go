package tests

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/ncruces/go-sqlite3/vfs/mvcc"
	_ "github.com/ncruces/go-sqlite3/vfs/xts"
)

func TestMain(m *testing.M) {
	sqlite3.Initialize()
	sqlite3.ConfigLog(func(code sqlite3.ExtendedErrorCode, msg string) {
		switch code {
		case sqlite3.NOTICE_RECOVER_WAL:
			// Wal "recovery" is expected.
			break
		case sqlite3.NOTICE_RECOVER_ROLLBACK:
			// Rollback journal recovery is an error.
			log.Panicf("%v (%d): %s", code, code, msg)
		default:
			log.Printf("%v (%d): %s", code, code, msg)
		}
	})
	os.Exit(m.Run())
}

func Test_parallel(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	var iter int
	if testing.Short() {
		iter = 1000
	} else {
		iter = 5000
	}

	name := "file:" +
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"
	createDB(t, name)
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func Test_wal(t *testing.T) {
	if !vfs.SupportsSharedMemory {
		t.Skip("skipping without shared memory")
	}

	var iter int
	if testing.Short() {
		iter = 1000
	} else {
		iter = 5000
	}

	name := "file:" +
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(wal)" +
		"&_pragma=synchronous(off)"
	createDB(t, name)
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func Test_memdb(t *testing.T) {
	var iter int
	if testing.Short() {
		iter = 1000
	} else {
		iter = 5000
	}

	name := memdb.TestDB(t, url.Values{
		"_pragma": {"busy_timeout(10000)"},
	})
	createDB(t, name)
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func Test_mvcc(t *testing.T) {
	var iter int
	if testing.Short() {
		iter = 1000
	} else {
		iter = 5000
	}

	mvcc.Create("test.db", "")
	name := "file:/test.db?vfs=mvcc" +
		"&_pragma=busy_timeout(10000)"
	createDB(t, name)
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func Test_adiantum(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	var iter int
	if testing.Short() {
		iter = 500
	} else {
		iter = 2500
	}

	name := "file:" +
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?vfs=adiantum" +
		"&_pragma=hexkey(e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855)" +
		"&_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"
	createDB(t, name)
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func Test_xts(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	var iter int
	if testing.Short() {
		iter = 500
	} else {
		iter = 2500
	}

	name := "file:" +
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?vfs=xts" +
		"&_pragma=hexkey(e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855)" +
		"&_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"
	createDB(t, name)
	testParallel(t, name, iter)
	testIntegrity(t, name)
}

func Test_MultiProcess_rollback(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	file := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("Test_MultiProcess_dbfile", file)

	name := "file:" + filepath.ToSlash(file) +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"
	createDB(t, name)

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(exe, append(os.Args[1:],
		"-test.v", "-test.count=1", "-test.run=Test_ChildProcess_rollback")...)
	out, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
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

func Test_ChildProcess_rollback(t *testing.T) {
	file := os.Getenv("Test_MultiProcess_dbfile")
	if file == "" || testing.Short() {
		t.SkipNow()
	}

	name := "file:" + filepath.ToSlash(file) +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"

	testParallel(t, name, 1000)
}

func Test_MultiProcess_wal(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	file := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("Test_MultiProcess_dbfile", file)

	name := "file:" + filepath.ToSlash(file) +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(wal)" +
		"&_pragma=synchronous(off)"
	createDB(t, name)

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(exe, append(os.Args[1:],
		"-test.v", "-test.count=1", "-test.run=Test_ChildProcess_wal")...)
	out, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
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

func Test_ChildProcess_wal(t *testing.T) {
	file := os.Getenv("Test_MultiProcess_dbfile")
	if file == "" || testing.Short() {
		t.SkipNow()
	}

	name := "file:" + filepath.ToSlash(file) +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(wal)" +
		"&_pragma=synchronous(off)"

	testParallel(t, name, 1000)
}

func Benchmark_parallel(b *testing.B) {
	if !vfs.SupportsSharedMemory {
		b.Skip("skipping without shared memory")
	}

	name := "file:" +
		filepath.Join(b.TempDir(), "test.db") +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(truncate)" +
		"&_pragma=synchronous(off)"
	createDB(b, name)

	b.ResetTimer()
	testParallel(b, name, b.N)
}

func Benchmark_wal(b *testing.B) {
	if !vfs.SupportsSharedMemory {
		b.Skip("skipping without shared memory")
	}

	name := "file:" +
		filepath.Join(b.TempDir(), "test.db") +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(wal)" +
		"&_pragma=synchronous(off)"
	createDB(b, name)

	b.ResetTimer()
	testParallel(b, name, b.N)
}

func Benchmark_memdb(b *testing.B) {
	name := memdb.TestDB(b, url.Values{
		"_pragma": {"busy_timeout(10000)"},
	})
	createDB(b, name)

	b.ResetTimer()
	testParallel(b, name, b.N)
}

func Benchmark_mvcc(b *testing.B) {
	mvcc.Create("test.db", "")
	name := "file:/test.db?vfs=mvcc" +
		"&_pragma=busy_timeout(10000)"
	createDB(b, name)

	b.ResetTimer()
	testParallel(b, name, b.N)
}

func createDB(t testing.TB, name string) {
	db, err := sqlite3.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}
}

func testParallel(t testing.TB, name string, n int) {
	writer := func() error {
		db, err := sqlite3.Open(name)
		if err != nil {
			return fmt.Errorf("writer: open: %w", err)
		}
		defer db.Close()

		err = db.Exec(`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
		if err != nil {
			return fmt.Errorf("writer: insert: %w", err)
		}

		return db.Close()
	}

	reader := func() error {
		db, err := sqlite3.Open(name)
		if err != nil {
			return fmt.Errorf("reader: open: %w", err)
		}
		defer db.Close()

		stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
		if err != nil {
			return fmt.Errorf("reader: select: %w", err)
		}
		defer stmt.Close()

		row := 0
		for stmt.Step() {
			row++
		}
		if err := stmt.Err(); err != nil {
			return fmt.Errorf("reader: step: %w", err)
		}
		if row%3 != 0 {
			return fmt.Errorf("reader: got %d rows, want multiple of 3", row)
		}

		err = stmt.Close()
		if err != nil {
			return fmt.Errorf("reader: close: %w", err)
		}

		return db.Close()
	}

	err := writer()
	if err != nil {
		t.Fatal(err)
	}

	var group errgroup.Group
	group.SetLimit(6)
	for i := range n {
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
