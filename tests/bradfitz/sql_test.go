//go:build (linux || darwin || windows || freebsd) && !sqlite3_nosys

package bradfitz

// Adapted from: https://github.com/bradfitz/go-sql-test

import (
	"database/sql"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/tests/testcfg"
)

type Tester interface {
	RunTest(*testing.T, func(params))
}

var (
	sqlite Tester = sqliteDB{}
)

const TablePrefix = "gosqltest_"

type sqliteDB struct{}

type params struct {
	dbType Tester
	*testing.T
	*sql.DB
}

func (t params) mustExec(sql string, args ...interface{}) sql.Result {
	res, err := t.DB.Exec(sql, args...)
	if err != nil {
		t.Fatalf("Error running %q: %v", sql, err)
	}
	return res
}

func (sqliteDB) RunTest(t *testing.T, fn func(params)) {
	db, err := sql.Open("sqlite3", "file:"+
		filepath.Join(t.TempDir(), "foo.db")+
		"?_pragma=busy_timeout(10000)&_pragma=synchronous(off)")
	if err != nil {
		t.Fatalf("foo.db open fail: %v", err)
	}
	fn(params{sqlite, t, db})
	if err := db.Close(); err != nil {
		t.Fatalf("foo.db close fail: %v", err)
	}
}

func TestBlobs_SQLite(t *testing.T) { sqlite.RunTest(t, testBlobs) }

func testBlobs(t params) {
	var blob = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	t.mustExec("create table " + TablePrefix + "foo (id integer primary key, bar blob)")
	t.mustExec("insert into "+TablePrefix+"foo (id, bar) values(?,?)", 0, blob)

	want := fmt.Sprintf("%x", blob)

	b := make([]byte, 16)
	err := t.QueryRow("select bar from "+TablePrefix+"foo where id = ?", 0).Scan(&b)
	got := fmt.Sprintf("%x", b)
	if err != nil {
		t.Errorf("[]byte scan: %v", err)
	} else if got != want {
		t.Errorf("for []byte, got %q; want %q", got, want)
	}

	err = t.QueryRow("select bar from "+TablePrefix+"foo where id = ?", 0).Scan(&got)
	want = string(blob)
	if err != nil {
		t.Errorf("string scan: %v", err)
	} else if got != want {
		t.Errorf("for string, got %q; want %q", got, want)
	}
}

func TestManyQueryRow_SQLite(t *testing.T) { sqlite.RunTest(t, testManyQueryRow) }

func testManyQueryRow(t params) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.mustExec("create table " + TablePrefix + "foo (id integer primary key, name varchar(50))")
	t.mustExec("insert into "+TablePrefix+"foo (id, name) values(?,?)", 1, "bob")
	var name string
	for i := 0; i < 10000; i++ {
		err := t.QueryRow("select name from "+TablePrefix+"foo where id = ?", 1).Scan(&name)
		if err != nil || name != "bob" {
			t.Fatalf("on query %d: err=%v, name=%q", i, err, name)
		}
	}
}

func TestTxQuery_SQLite(t *testing.T) { sqlite.RunTest(t, testTxQuery) }

func testTxQuery(t params) {
	tx, err := t.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("create table " + TablePrefix + "foo (id integer primary key, name varchar(50))")
	if err != nil {
		t.Logf("cannot drop table "+TablePrefix+"foo: %s", err)
	}

	_, err = tx.Exec("insert into "+TablePrefix+"foo (id, name) values(?,?)", 1, "bob")
	if err != nil {
		t.Fatal(err)
	}

	r, err := tx.Query("select name from "+TablePrefix+"foo where id = ?", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if !r.Next() {
		if r.Err() != nil {
			t.Fatal(err)
		}
		t.Fatal("expected one row")
	}

	var name string
	err = r.Scan(&name)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreparedStmt_SQLite(t *testing.T) { sqlite.RunTest(t, testPreparedStmt) }

func testPreparedStmt(t params) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.mustExec("CREATE TABLE " + TablePrefix + "t (count INT)")
	sel, err := t.Prepare("SELECT count FROM " + TablePrefix + "t ORDER BY count DESC")
	if err != nil {
		t.Fatalf("prepare 1: %v", err)
	}
	ins, err := t.Prepare("INSERT INTO " + TablePrefix + "t (count) VALUES (?)")
	if err != nil {
		t.Fatalf("prepare 2: %v", err)
	}

	for n := 1; n <= 3; n++ {
		if _, err := ins.Exec(n); err != nil {
			t.Fatalf("insert(%d) = %v", n, err)
		}
	}

	const nRuns = 10
	var wg sync.WaitGroup
	for i := 0; i < nRuns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				count := 0
				if err := sel.QueryRow().Scan(&count); err != nil && err != sql.ErrNoRows {
					t.Errorf("Query: %v", err)
					return
				}
				if _, err := ins.Exec(rand.Intn(100)); err != nil {
					t.Errorf("Insert: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
