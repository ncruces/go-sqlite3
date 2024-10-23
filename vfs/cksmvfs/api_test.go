package cksmvfs_test

import (
	_ "embed"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/cksmvfs"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
)

//go:embed testdata/test.db
var testDB string

func Test_fileformat(t *testing.T) {
	readervfs.Create("test.db", ioutil.NewSizeReaderAt(strings.NewReader(testDB)))
	cksmvfs.Register("rcksm", vfs.Find("reader"))

	db, err := driver.Open("file:test.db?vfs=rcksm")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Error(err)
	}
}

func Test_new(t *testing.T) {
	name := "file:" +
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?vfs=cksmvfs&_pragma=journal_mode(wal)"

	db, err := driver.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users (id INT, name VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxIdleConns(0)

	_, err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Error(err)
	}
}
