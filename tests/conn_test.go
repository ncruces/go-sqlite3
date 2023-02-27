package tests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestConn_Open_dir(t *testing.T) {
	t.Parallel()

	_, err := sqlite3.Open(".")
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}

func TestConn_Close(t *testing.T) {
	var conn *sqlite3.Conn
	conn.Close()
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
		t.Error("got message: ", got)
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
	cancel()

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
		t.Error("got message: ", got)
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
		t.Error("got SQL: ", got)
	}
	if got := serr.Error(); got != `sqlite3: SQL logic error: near "FRM": syntax error` {
		t.Error("got message: ", got)
	}
}

func TestConn_MustPrepare_empty(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	defer func() { _ = recover() }()
	stmt := db.MustPrepare(``)
	defer stmt.Close()

	if stmt != nil {
		t.Error("want nil")
	}
}

func TestConn_MustPrepare_tail(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	defer func() { _ = recover() }()
	_ = db.MustPrepare(`SELECT 1; -- HERE`)
	t.Error("want panic")
}

func TestConn_MustPrepare_invalid(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	defer func() { _ = recover() }()
	_ = db.MustPrepare(`SELECT`)
	t.Error("want panic")
}
