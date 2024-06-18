package bloom_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/bloom"
)

//go:embed testdata/bloom.db
var testDB []byte

func TestRegister(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "bloom.db")
	err := os.WriteFile(tmp, testDB, 0666)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sqlite3.Open("file:" + filepath.ToSlash(tmp) + "?nolock=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	bloom.Register(db)

	query, _, err := db.Prepare(`SELECT COUNT(*) FROM plants(?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()

	err = query.BindText(1, "apple")
	if err != nil {
		t.Fatal(err)
	}
	if !query.Step() {
		t.Error("no rows")
	}
	if !query.ColumnBool(0) {
		t.Error("want true")
	}
	err = query.Reset()
	if err != nil {
		t.Fatal(err)
	}

	err = query.BindText(1, "lemon")
	if err != nil {
		t.Fatal(err)
	}
	if !query.Step() {
		t.Error("no rows")
	}
	if query.ColumnBool(0) {
		t.Error("want false")
	}
	err = query.Reset()
	if err != nil {
		t.Fatal(err)
	}
}
