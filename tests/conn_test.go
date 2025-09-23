package tests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestConn_Open_dir(t *testing.T) {
	t.Parallel()

	_, err := sqlite3.OpenFlags(".", 0)
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}

func TestConn_Open_notfound(t *testing.T) {
	t.Parallel()

	_, err := sqlite3.OpenFlags("test.db", sqlite3.OPEN_READONLY)
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}

func TestConn_Open_modeof(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "test.db")
	mode := filepath.Join(dir, "modeof.txt")

	fd, err := os.OpenFile(mode, os.O_CREATE, 0624)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := fd.Stat()
	if err != nil {
		t.Fatal(err)
	}
	fd.Close()

	db, err := sqlite3.Open("file:" + file + "?modeof=" + mode)
	if err != nil {
		t.Fatal(err)
	}
	di, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	if di.Mode() != fi.Mode() {
		t.Errorf("got %v, want %v", di.Mode(), fi.Mode())
	}

	_, err = sqlite3.Open("file:" + file + "?modeof=" + mode + "2")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestConn_Close(t *testing.T) {
	var db *sqlite3.Conn
	db.Close()
}

func TestConn_Close_BUSY(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`BEGIN`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	err = db.Close()
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.Is(err, sqlite3.BUSY) {
		t.Errorf("got %v, want sqlite3.BUSY", err)
	}
	var terr interface{ Temporary() bool }
	if !errors.As(err, &terr) || !terr.Temporary() {
		t.Error("not temporary", err)
	}
	if got := err.Error(); got != `sqlite3: database is locked: unable to close due to unfinalized statements or unfinished backups` {
		t.Error("got message:", got)
	}
}

func TestConn_BusyHandler(t *testing.T) {
	t.Parallel()

	dsn := memdb.TestDB(t)

	db1, err := sqlite3.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()

	db2, err := sqlite3.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	var called bool
	err = db2.BusyHandler(func(ctx context.Context, count int) (retry bool) {
		called = true
		return count < 1
	})
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db1.BeginExclusive()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.End(&err)

	_, err = db2.BeginExclusive()
	if !errors.Is(err, sqlite3.BUSY) {
		t.Errorf("got %v, want sqlite3.BUSY", err)
	}

	if !called {
		t.Error("busy handler not called")
	}
}

func TestConn_SetInterrupt(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(t.Context())
	db.SetInterrupt(ctx)

	// Interrupt doesn't interrupt this.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		WITH RECURSIVE
		  fibonacci (curr, next)
		AS (
		  SELECT 0, 1
		  UNION ALL
		  SELECT next, curr + next FROM fibonacci
		  LIMIT 1e7
		)
		SELECT min(curr) FROM fibonacci
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	time.AfterFunc(time.Millisecond, cancel)

	// Interrupting works.
	err = stmt.Exec()
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	// Interrupting sticks.
	err = db.Exec(`SELECT 1`)
	if !errors.Is(err, sqlite3.INTERRUPT) {
		t.Errorf("got %v, want sqlite3.INTERRUPT", err)
	}

	db.SetInterrupt(t.Context())

	// Interrupting can be cleared.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		t.Fatal(err)
	}

	if got := db.GetInterrupt(); got != t.Context() {
		t.Errorf("got %v, want %v", got, t.Context())
	}
}

func TestConn_Prepare_empty(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(``)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt != nil {
		t.Error("want nil")
	}
}

func TestConn_Prepare_tail(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, tail, err := db.Prepare(`SELECT 1; -- HERE`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if !strings.Contains(tail, "-- HERE") {
		t.Errorf("got %q", tail)
	}
}

func TestConn_Prepare_invalid(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var serr *sqlite3.Error

	_, _, err = db.Prepare(`SELECT`)
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.As(err, &serr) {
		t.Fatalf("got %T, want sqlite3.Error", err)
	}
	if rc := serr.Code(); rc != sqlite3.ERROR {
		t.Errorf("got %d, want sqlite3.ERROR", rc)
	}
	if got := err.Error(); got != `sqlite3: SQL logic error: incomplete input` {
		t.Error("got message:", got)
	}

	_, _, err = db.Prepare(`SELECT * FRM sqlite_schema`)
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.As(err, &serr) {
		t.Fatalf("got %T, want sqlite3.ERROR", err)
	}
	if rc := serr.Code(); rc != sqlite3.ERROR {
		t.Errorf("got %d, want sqlite3.ERROR", rc)
	}
	if got := serr.SQL(); got != `FRM sqlite_schema` {
		t.Error("got SQL:", got)
	}
	if got := serr.Error(); got != `sqlite3: SQL logic error: near "FRM": syntax error` {
		t.Error("got message:", got)
	}
}

func TestConn_ReleaseMemory(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.ReleaseMemory()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConn_SetLastInsertRowID(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetLastInsertRowID(42)

	got := db.LastInsertRowID()
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestConn_Filename(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "test.db")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	file, err = filepath.EvalSymlinks(file)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sqlite3.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	n := db.Filename("")
	if n.String() != file {
		t.Errorf("got %v", n)
	}
	if n.Database() != file {
		t.Errorf("got %v", n)
	}
	if n.DatabaseFile() == nil {
		t.Errorf("got %v", n)
	}

	n = db.Filename("xpto")
	if n != nil {
		t.Errorf("got %v", n)
	}
	if n.String() != "" {
		t.Errorf("got %v", n)
	}
	if n.Database() != "" {
		t.Errorf("got %v", n)
	}
	if n.Journal() != "" {
		t.Errorf("got %v", n)
	}
	if n.WAL() != "" {
		t.Errorf("got %v", n)
	}
	if n.DatabaseFile() != nil {
		t.Errorf("got %v", n)
	}
}

func TestConn_ReadOnly(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if ro, ok := db.ReadOnly(""); ro != false || ok != false {
		t.Errorf("got %v,%v", ro, ok)
	}

	if ro, ok := db.ReadOnly("xpto"); ro != false || ok != true {
		t.Errorf("got %v,%v", ro, ok)
	}
}

func TestConn_DBName(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if name := db.DBName(0); name != "main" {
		t.Errorf("got %s", name)
	}

	if name := db.DBName(5); name != "" {
		t.Errorf("got %s", name)
	}
}

func TestConn_Status(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	cr, hi, err := db.Status(sqlite3.DBSTATUS_SCHEMA_USED, true)
	if err != nil {
		t.Error("want nil")
	}
	if cr == 0 {
		t.Error("want something")
	}
	if hi != 0 {
		t.Error("want zero")
	}

	cr, hi, err = db.Status(sqlite3.DBSTATUS_LOOKASIDE_HIT, true)
	if err != nil {
		t.Error("want nil")
	}
	if cr != 0 {
		t.Error("want zero")
	}
	if hi == 0 {
		t.Error("want something")
	}

	cr, hi, err = db.Status(sqlite3.DBSTATUS_LOOKASIDE_HIT, true)
	if err != nil {
		t.Error("want nil")
	}
	if cr != 0 {
		t.Error("want zero")
	}
	if hi != 0 {
		t.Error("want zero")
	}
}

func TestConn_TableColumnMetadata(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, _, _, err = db.TableColumnMetadata("", "table", "")
	if err == nil {
		t.Error("want error")
	}

	_, _, _, _, _, err = db.TableColumnMetadata("", "test", "")
	if err != nil {
		t.Error("want nil")
	}

	typ, ord, nn, pk, ai, err := db.TableColumnMetadata("main", "test", "rowid")
	if err != nil {
		t.Error("want nil")
	}
	if typ != "INTEGER" {
		t.Error("want INTEGER")
	}
	if ord != "BINARY" {
		t.Error("want BINARY")
	}
	if nn != false {
		t.Error("want false")
	}
	if pk != true {
		t.Error("want true")
	}
	if ai != false {
		t.Error("want false")
	}
}
