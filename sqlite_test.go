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

func Test_sqlite_error_OOM(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	defer func() { _ = recover() }()
	sqlite.error(uint64(NOMEM), 0)
	t.Error("want panic")
}

func Test_sqlite_call_closed(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	sqlite.close()

	defer func() { _ = recover() }()
	sqlite.call(sqlite.api.free)
	t.Error("want panic")
}

func Test_sqlite_new(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	t.Run("MaxUint32", func(t *testing.T) {
		defer func() { _ = recover() }()
		sqlite.new(math.MaxUint32)
		t.Error("want panic")
	})

	t.Run("_MAX_ALLOCATION_SIZE", func(t *testing.T) {
		defer func() { _ = recover() }()
		sqlite.new(_MAX_ALLOCATION_SIZE)
		sqlite.new(_MAX_ALLOCATION_SIZE)
		t.Error("want panic")
	})
}

func Test_sqlite_newArena(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	arena := sqlite.newArena(16)
	defer arena.free()

	const title = "Lorem ipsum"
	ptr := arena.string(title)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := util.ReadString(sqlite.mod, ptr, math.MaxUint32); got != title {
		t.Errorf("got %q, want %q", got, title)
	}

	const body = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
	ptr = arena.string(body)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := util.ReadString(sqlite.mod, ptr, math.MaxUint32); got != body {
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
	if got := util.View(sqlite.mod, ptr, uint64(len(title))); string(got) != title {
		t.Errorf("got %q, want %q", got, title)
	}

	arena.free()
}

func Test_sqlite_newBytes(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	ptr := sqlite.newBytes(nil)
	if ptr != 0 {
		t.Errorf("got %#x, want nullptr", ptr)
	}

	buf := []byte("sqlite3")
	ptr = sqlite.newBytes(buf)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := buf
	if got := util.View(sqlite.mod, ptr, uint64(len(want))); !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	ptr = sqlite.newBytes(buf[:0])
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	if got := util.View(sqlite.mod, ptr, 0); got != nil {
		t.Errorf("got %q, want nil", got)
	}
}

func Test_sqlite_newString(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	ptr := sqlite.newString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3\000sqlite3"
	ptr = sqlite.newString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := str + "\000"
	if got := util.View(sqlite.mod, ptr, uint64(len(want))); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_sqlite_getString(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	ptr := sqlite.newString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3" + "\000 drop this"
	ptr = sqlite.newString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := "sqlite3"
	if got := util.ReadString(sqlite.mod, ptr, math.MaxUint32); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := util.ReadString(sqlite.mod, ptr, 0); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	func() {
		defer func() { _ = recover() }()
		util.ReadString(sqlite.mod, ptr, uint32(len(want)/2))
		t.Error("want panic")
	}()

	func() {
		defer func() { _ = recover() }()
		util.ReadString(sqlite.mod, 0, math.MaxUint32)
		t.Error("want panic")
	}()
}

func Test_sqlite_free(t *testing.T) {
	t.Parallel()

	sqlite, err := instantiateSQLite()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.close()

	sqlite.free(0)

	ptr := sqlite.new(1)
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	sqlite.free(ptr)
}
