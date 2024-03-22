package tests

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
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

func TestConn_SetInterrupt(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	db.SetInterrupt(ctx)

	// Interrupt doesn't interrupt this.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		t.Fatal(err)
	}

	db.SetInterrupt(context.Background())

	stmt, _, err := db.Prepare(`
		WITH RECURSIVE
		  fibonacci (curr, next)
		AS (
		  SELECT 0, 1
		  UNION ALL
		  SELECT next, curr + next FROM fibonacci
		  LIMIT 1e6
		)
		SELECT min(curr) FROM fibonacci
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	db.SetInterrupt(ctx)
	go cancel()

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

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	db.SetInterrupt(ctx)

	// Interrupting can be cleared.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		t.Fatal(err)
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

func TestConn_Config(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	o, err := db.Config(sqlite3.DBCONFIG_DEFENSIVE)
	if err != nil {
		t.Fatal(err)
	}
	if o != false {
		t.Error("want false")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE, true)
	if err != nil {
		t.Fatal(err)
	}
	if o != true {
		t.Error("want true")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE)
	if err != nil {
		t.Fatal(err)
	}
	if o != true {
		t.Error("want true")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE, false)
	if err != nil {
		t.Fatal(err)
	}
	if o != false {
		t.Error("want false")
	}

	o, err = db.Config(sqlite3.DBCONFIG_DEFENSIVE)
	if err != nil {
		t.Fatal(err)
	}
	if o != false {
		t.Error("want false")
	}
}

func TestConn_ConfigLog(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var code sqlite3.ExtendedErrorCode
	err = db.ConfigLog(func(c sqlite3.ExtendedErrorCode, msg string) {
		t.Log(msg)
		code = c
	})
	if err != nil {
		t.Fatal(err)
	}

	db.Prepare(`SELECT * FRM sqlite_schema`)

	if code != sqlite3.ExtendedErrorCode(sqlite3.ERROR) {
		t.Error("want sqlite3.ERROR")
	}
}

func TestConn_Limit(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	l := db.Limit(sqlite3.LIMIT_COLUMN, -1)
	if l != 2000 {
		t.Errorf("got %d, want 2000", l)
	}

	l = db.Limit(sqlite3.LIMIT_COLUMN, 100)
	if l != 2000 {
		t.Errorf("got %d, want 2000", l)
	}

	l = db.Limit(sqlite3.LIMIT_COLUMN, -1)
	if l != 100 {
		t.Errorf("got %d, want 100", l)
	}

	l = db.Limit(math.MaxUint32, -1)
	if l != -1 {
		t.Errorf("got %d, want -1", l)
	}
}

func TestConn_SetAuthorizer(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.SetAuthorizer(func(action sqlite3.AuthorizerActionCode, name3rd, name4th, schema, nameInner string) sqlite3.AuthorizerReturnCode {
		return sqlite3.AUTH_DENY
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`SELECT * FROM sqlite_schema`)
	if !errors.Is(err, sqlite3.AUTH) {
		t.Errorf("got %v, want sqlite3.AUTH", err)
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
