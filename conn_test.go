package sqlite3

import (
	"bytes"
	"math"
	"testing"
)

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
	defer arena.free()

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
	arena.free()
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
