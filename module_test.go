package sqlite3

import (
	"bytes"
	"math"
	"testing"
)

func TestConn_error_OOM(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	defer func() { _ = recover() }()
	m.error(uint64(NOMEM), 0)
	t.Error("want panic")
}

func TestConn_call_nil(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	defer func() { _ = recover() }()
	m.call(m.api.free)
	t.Error("want panic")
}

func TestConn_new(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	testOOM := func(size uint64) {
		defer func() { _ = recover() }()
		m.new(size)
		t.Error("want panic")
	}

	testOOM(math.MaxUint32)
	testOOM(_MAX_ALLOCATION_SIZE)
}

func TestConn_newArena(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	arena := m.newArena(16)
	defer arena.free()

	const title = "Lorem ipsum"

	ptr := arena.string(title)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := m.mem.readString(ptr, math.MaxUint32); got != title {
		t.Errorf("got %q, want %q", got, title)
	}

	const body = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
	ptr = arena.string(body)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := m.mem.readString(ptr, math.MaxUint32); got != body {
		t.Errorf("got %q, want %q", got, body)
	}
	arena.free()
}

func TestConn_newBytes(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	ptr := m.newBytes(nil)
	if ptr != 0 {
		t.Errorf("got %#x, want nullptr", ptr)
	}

	buf := []byte("sqlite3")
	ptr = m.newBytes(buf)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := buf
	if got := m.mem.view(ptr, uint64(len(want))); !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConn_newString(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	ptr := m.newString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3\000sqlite3"
	ptr = m.newString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := str + "\000"
	if got := m.mem.view(ptr, uint64(len(want))); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConn_getString(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	ptr := m.newString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3" + "\000 drop this"
	ptr = m.newString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := "sqlite3"
	if got := m.mem.readString(ptr, math.MaxUint32); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := m.mem.readString(ptr, 0); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	func() {
		defer func() { _ = recover() }()
		m.mem.readString(ptr, uint32(len(want)/2))
		t.Error("want panic")
	}()

	func() {
		defer func() { _ = recover() }()
		m.mem.readString(0, math.MaxUint32)
		t.Error("want panic")
	}()
}

func TestConn_free(t *testing.T) {
	t.Parallel()

	m, err := instantiateModule()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	m.free(0)

	ptr := m.new(1)
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	m.free(ptr)
}
