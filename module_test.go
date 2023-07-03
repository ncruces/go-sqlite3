package sqlite3

import (
	"bytes"
	"math"
	"testing"

	"github.com/ncruces/go-sqlite3/internal/util"
)

func init() {
	Path = "./embed/sqlite3.wasm"
}

func TestConn_error_OOM(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	defer func() { _ = recover() }()
	m.error(uint64(NOMEM), 0)
	t.Error("want panic")
}

func TestConn_call_closed(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	m.close()

	defer func() { _ = recover() }()
	m.call(m.api.free)
	t.Error("want panic")
}

func TestConn_new(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer m.close()

	t.Run("MaxUint32", func(t *testing.T) {
		defer func() { _ = recover() }()
		m.new(math.MaxUint32)
		t.Error("want panic")
	})

	t.Run("_MAX_ALLOCATION_SIZE", func(t *testing.T) {
		defer func() { _ = recover() }()
		m.new(_MAX_ALLOCATION_SIZE)
		m.new(_MAX_ALLOCATION_SIZE)
		t.Error("want panic")
	})
}

func TestConn_newArena(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
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
	if got := util.ReadString(m.mod, ptr, math.MaxUint32); got != title {
		t.Errorf("got %q, want %q", got, title)
	}

	const body = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
	ptr = arena.string(body)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := util.ReadString(m.mod, ptr, math.MaxUint32); got != body {
		t.Errorf("got %q, want %q", got, body)
	}

	ptr = arena.bytes(nil)
	if ptr != 0 {
		t.Errorf("want nullptr")
	}
	ptr = arena.bytes([]byte(title))
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := util.View(m.mod, ptr, uint64(len(title))); string(got) != title {
		t.Errorf("got %q, want %q", got, title)
	}

	arena.free()
}

func TestConn_newBytes(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
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
	if got := util.View(m.mod, ptr, uint64(len(want))); !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConn_newString(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
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
	if got := util.View(m.mod, ptr, uint64(len(want))); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConn_getString(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
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
	if got := util.ReadString(m.mod, ptr, math.MaxUint32); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := util.ReadString(m.mod, ptr, 0); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	func() {
		defer func() { _ = recover() }()
		util.ReadString(m.mod, ptr, uint32(len(want)/2))
		t.Error("want panic")
	}()

	func() {
		defer func() { _ = recover() }()
		util.ReadString(m.mod, 0, math.MaxUint32)
		t.Error("want panic")
	}()
}

func TestConn_free(t *testing.T) {
	t.Parallel()

	m, err := instantiateSQLite()
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
