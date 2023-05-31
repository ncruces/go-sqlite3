package tests

import (
	"errors"
	"strings"
	"testing"

	_ "embed"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

//go:embed testdata/test.db
var testdata string

func TestMemoryVFS_Open_notfound(t *testing.T) {
	sqlite3vfs.Register("memory", sqlite3vfs.MemoryVFS{
		"test.db": &sqlite3vfs.MemoryDB{},
	})
	defer sqlite3vfs.Unregister("memory")

	_, err := sqlite3.Open("file:demo.db?vfs=memory&mode=ro")
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}

func TestMemoryVFS_Open_errors(t *testing.T) {
	sqlite3vfs.Register("memory", sqlite3vfs.MemoryVFS{
		"test.db": &sqlite3vfs.MemoryDB{MaxSize: 65536},
	})
	defer sqlite3vfs.Unregister("memory")

	db, err := sqlite3.Open("file:test.db?vfs=memory")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO test VALUES (zeroblob(65536))`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.FULL) {
		t.Errorf("got %v, want sqlite3.FULL", err)
	}
}

func TestReaderVFS_Open_notfound(t *testing.T) {
	sqlite3vfs.Register("reader", sqlite3vfs.ReaderVFS{
		"test.db": sqlite3vfs.NewSizeReaderAt(strings.NewReader(testdata)),
	})
	defer sqlite3vfs.Unregister("reader")

	_, err := sqlite3.Open("file:demo.db?vfs=reader&mode=ro")
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}
