package serdes_test

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/serdes"
)

func TestDeserialize(t *testing.T) {
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

	if len(input) != len(output) {
		t.Fatal("lengths are different")
	}
	for i := range input {
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
		if input[i] != output[i] {
			t.Errorf("difference at %d: %d %d", i, input[i], output[i])
		}
	}
}

func httpGet() ([]byte, error) {
	res, err := http.Get("https://raw.githubusercontent.com/jpwhite3/northwind-SQLite3/refs/heads/main/dist/northwind.db")
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
