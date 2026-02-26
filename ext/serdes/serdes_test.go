package serdes_test

import (
	_ "embed"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/serdes"
)

//go:embed testdata/wal.db
var walDB []byte

func Test_wal(t *testing.T) {
	db, err := sqlite3.Open("testdata/wal.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	data, err := serdes.Serialize(db, "main")
	if err != nil {
		t.Fatal(err)
	}

	compareDBs(t, data, walDB)

	err = serdes.Deserialize(db, "temp", walDB)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_northwind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	input, err := httpGet()
	if err != nil {
		t.Fatal(err)
	}

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = serdes.Deserialize(db, "temp", input)
	if err != nil {
		t.Fatal(err)
	}

	output, err := serdes.Serialize(db, "temp")
	if err != nil {
		t.Fatal(err)
	}

	compareDBs(t, input, output)
}

func compareDBs(t *testing.T, a, b []byte) {
	if len(a) != len(b) {
		t.Fatal("lengths are different")
	}
	for i := range a {
		// These may be different.
		switch {
		case 24 <= i && i < 28:
			// File change counter.
			continue
		case 40 <= i && i < 44:
			// Schema cookie.
			continue
		case 92 <= i && i < 100:
			// SQLite version that wrote the file.
			continue
		}
		if a[i] != b[i] {
			t.Errorf("difference at %d: %d %d", i, a[i], b[i])
		}
	}
}

func httpGet() ([]byte, error) {
	res, err := http.Get("https://github.com/jpwhite3/northwind-SQLite3/raw/refs/heads/main/dist/northwind.db")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func TestOpen_errors(t *testing.T) {
	_, err := sqlite3.Open("file:test.db?vfs=github.com/ncruces/go-sqlite3/ext/serdes.sliceVFS")
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}

	_, err = sqlite3.Open("file:serdes.db?vfs=github.com/ncruces/go-sqlite3/ext/serdes.sliceVFS")
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.MISUSE) {
		t.Errorf("got %v, want sqlite3.MISUSE", err)
	}
}
