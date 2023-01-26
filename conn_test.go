package sqlite3

import (
	"bytes"
	"math"
	"testing"
)

func TestConn_new(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	defer func() { _ = recover() }()
	db.new(math.MaxUint32)
	t.Errorf("should have panicked")
}

func TestConn_newBytes(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ptr := db.newBytes(nil)
	if ptr != 0 {
		t.Errorf("want nullptr got %x", ptr)
	}

	buf := []byte("sqlite3")
	ptr = db.newBytes(buf)
	if ptr == 0 {
		t.Errorf("want a pointer got nullptr")
	}

	want := buf
	if got, ok := db.memory.Read(ptr, uint32(len(want))); !ok || !bytes.Equal(want, got) {
		t.Errorf("want %q got %q", want, got)
	}
}

func TestConn_newString(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ptr := db.newString("")
	if ptr == 0 {
		t.Errorf("want a pointer got nullptr")
	}

	str := "sqlite3\000sqlite3"
	ptr = db.newString(str)
	if ptr == 0 {
		t.Errorf("want a pointer got nullptr")
	}

	want := str + "\000"
	if got, ok := db.memory.Read(ptr, uint32(len(want))); !ok || want != string(got) {
		t.Errorf("want %q got %q", want, got)
	}
}

func TestConn_getString(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ptr := db.newString("")
	if ptr == 0 {
		t.Errorf("want a pointer got nullptr")
	}

	str := "sqlite3" + "\000 drop this"
	ptr = db.newString(str)
	if ptr == 0 {
		t.Errorf("want a pointer got nullptr")
	}

	want := "sqlite3"
	if got := db.getString(ptr, math.MaxUint32); want != got {
		t.Errorf("want %q got %q", want, got)
	}
	if got := db.getString(ptr, 0); got != "" {
		t.Errorf("want empty got %q", got)
	}

	func() {
		defer func() { _ = recover() }()
		db.getString(ptr, uint32(len(want)/2))
		t.Errorf("should have panicked")
	}()

	func() {
		defer func() { _ = recover() }()
		db.getString(0, math.MaxUint32)
		t.Errorf("should have panicked")
	}()
}

func TestConn_free(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.free(0)

	ptr := db.new(0)
	if ptr == 0 {
		t.Errorf("want a pointer got nullptr")
	}

	db.free(ptr)
}
