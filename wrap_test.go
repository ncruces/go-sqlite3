package sqlite3

import (
	"bytes"
	"context"
	"math"
	"testing"
)

func Test_sqlite_new(t *testing.T) {
	t.Parallel()

	wrp, err := createWrapper(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	t.Run("MaxUint32", func(t *testing.T) {
		defer func() { _ = recover() }()
		wrp.New(math.MaxUint32)
		t.Error("want panic")
	})
}

func Test_sqlite_newArena(t *testing.T) {
	t.Parallel()

	wrp, err := createWrapper(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	arena := wrp.NewArena()
	defer arena.Free()

	const title = "Lorem ipsum"
	ptr := arena.String(title)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := wrp.ReadString(ptr, math.MaxInt); got != title {
		t.Errorf("got %q, want %q", got, title)
	}

	const body = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
	ptr = arena.String(body)
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := wrp.ReadString(ptr, math.MaxInt); got != body {
		t.Errorf("got %q, want %q", got, body)
	}

	ptr = arena.Bytes(nil)
	if ptr != 0 {
		t.Errorf("want nullptr")
	}
	ptr = arena.Bytes([]byte(title))
	if ptr == 0 {
		t.Fatalf("got nullptr")
	}
	if got := wrp.Bytes(ptr, int64(len(title))); string(got) != title {
		t.Errorf("got %q, want %q", got, title)
	}

	arena.Free()
}

func Test_sqlite_newBytes(t *testing.T) {
	t.Parallel()

	wrp, err := createWrapper(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	ptr := wrp.NewBytes(nil)
	if ptr != 0 {
		t.Errorf("got %#x, want nullptr", ptr)
	}

	buf := []byte("sqlite3")
	ptr = wrp.NewBytes(buf)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := buf
	if got := wrp.Bytes(ptr, int64(len(want))); !bytes.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_sqlite_newString(t *testing.T) {
	t.Parallel()

	wrp, err := createWrapper(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	ptr := wrp.NewString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3\000sqlite3"
	ptr = wrp.NewString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := str + "\000"
	if got := wrp.Bytes(ptr, int64(len(want))); string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_sqlite_getString(t *testing.T) {
	t.Parallel()

	wrp, err := createWrapper(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	ptr := wrp.NewString("")
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	str := "sqlite3" + "\000 drop this"
	ptr = wrp.NewString(str)
	if ptr == 0 {
		t.Fatal("got nullptr, want a pointer")
	}

	want := "sqlite3"
	if got := wrp.ReadString(ptr, math.MaxInt); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := wrp.ReadString(ptr, 0); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	func() {
		defer func() { _ = recover() }()
		wrp.ReadString(ptr, int64(len(want))/2)
		t.Error("want panic")
	}()

	func() {
		defer func() { _ = recover() }()
		wrp.ReadString(0, math.MaxInt)
		t.Error("want panic")
	}()
}

func Test_sqlite_free(t *testing.T) {
	t.Parallel()

	wrp, err := createWrapper(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	defer wrp.Close()

	wrp.Free(0)

	ptr := wrp.New(1)
	if ptr == 0 {
		t.Error("got nullptr, want a pointer")
	}

	wrp.Free(ptr)
}

func testContext(t testing.TB) context.Context {
	return WithMaxMemory(t.Context(), 32*1024*1024)
}
