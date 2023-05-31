package tests

import (
	"errors"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/sqlite3memdb"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

func TestMemoryVFS_Open_notfound(t *testing.T) {
	sqlite3memdb.Delete("demo.db")

	_, err := sqlite3.Open("file:/demo.db?vfs=memdb&mode=ro")
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}

func TestReaderVFS_Open_notfound(t *testing.T) {
	sqlite3vfs.Register("reader", sqlite3vfs.ReaderVFS{})
	defer sqlite3vfs.Unregister("reader")

	_, err := sqlite3.Open("file:demo.db?vfs=reader&mode=ro")
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}
