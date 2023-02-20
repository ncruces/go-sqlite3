package sqlite3

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestConn_Close(t *testing.T) {
	var conn *Conn
	conn.Close()
}

func TestConn_Close_BUSY(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
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
	var serr *Error
	if !errors.As(err, &serr) {
		t.Fatalf("got %T, want sqlite3.Error", err)
	}
	if rc := serr.Code(); rc != BUSY {
		t.Errorf("got %d, want sqlite3.BUSY", rc)
	}
	if got := err.Error(); got != `sqlite3: database is locked: unable to close due to unfinalized statements or unfinished backups` {
		t.Error("got message: ", got)
	}
}

func TestConn_SetInterrupt(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	db.SetInterrupt(ctx.Done())

	// Interrupt doesn't interrupt this.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		t.Fatal(err)
	}

	db.SetInterrupt(nil)

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

	cancel()
	db.SetInterrupt(ctx.Done())

	var serr *Error

	// Interrupting works.
	err = stmt.Exec()
	if err != nil {
		if !errors.As(err, &serr) {
			t.Fatalf("got %T, want sqlite3.Error", err)
		}
		if rc := serr.Code(); rc != INTERRUPT {
			t.Errorf("got %d, want sqlite3.INTERRUPT", rc)
		}
		if got := err.Error(); got != `sqlite3: interrupted` {
			t.Error("got message: ", got)
		}
	}

	// Interrupting sticks.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		if !errors.As(err, &serr) {
			t.Fatalf("got %T, want sqlite3.Error", err)
		}
		if rc := serr.Code(); rc != INTERRUPT {
			t.Errorf("got %d, want sqlite3.INTERRUPT", rc)
		}
		if got := err.Error(); got != `sqlite3: interrupted` {
			t.Error("got message: ", got)
		}
	}

	db.SetInterrupt(nil)

	// Interrupting can be cleared.
	err = db.Exec(`SELECT 1`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConn_Prepare_Empty(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
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

func TestConn_Prepare_Tail(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
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

func TestConn_Prepare_Invalid(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var serr *Error

	_, _, err = db.Prepare(`SELECT`)
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.As(err, &serr) {
		t.Fatalf("got %T, want sqlite3.Error", err)
	}
	if rc := serr.Code(); rc != ERROR {
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
	if rc := serr.Code(); rc != ERROR {
		t.Errorf("got %d, want sqlite3.ERROR", rc)
	}
	if got := serr.SQL(); got != `FRM sqlite_schema` {
		t.Error("got SQL: ", got)
	}
	if got := serr.Error(); got != `sqlite3: SQL logic error: near "FRM": syntax error` {
		t.Error("got message: ", got)
	}
}

func TestConn_new(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	defer func() { _ = recover() }()
	db.new(math.MaxUint32)
	t.Error("want panic")
}

func TestConn_newArena(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	arena := db.newArena(16)
	defer arena.reset()

	const title = "Lorem ipsum"

	ptr := arena.string(title)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := db.mem.readString(ptr, math.MaxUint32); got != title {
		t.Errorf("got %q, want %q", got, title)
	}

	const body = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
	ptr = arena.string(body)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := db.mem.readString(ptr, math.MaxUint32); got != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestConn_newBytes(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ptr := db.newBytes(nil)
	if ptr != 0 {
		t.Errorf("got %#x, want nullptr", ptr)
	}

	buf := []byte("sqlite3")
	ptr = db.newBytes(buf)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := buf
	if got := db.mem.view(ptr, uint32(len(want))); !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConn_newString(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ptr := db.newString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3\000sqlite3"
	ptr = db.newString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := str + "\000"
	if got := db.mem.view(ptr, uint32(len(want))); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConn_getString(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ptr := db.newString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3" + "\000 drop this"
	ptr = db.newString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := "sqlite3"
	if got := db.mem.readString(ptr, math.MaxUint32); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := db.mem.readString(ptr, 0); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	func() {
		defer func() { _ = recover() }()
		db.mem.readString(ptr, uint32(len(want)/2))
		t.Error("want panic")
	}()

	func() {
		defer func() { _ = recover() }()
		db.mem.readString(0, math.MaxUint32)
		t.Error("want panic")
	}()
}

func TestConn_free(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.free(0)

	ptr := db.new(1)
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	db.free(ptr)
}
