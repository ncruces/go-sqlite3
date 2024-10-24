package cksmvfs_test

import (
	_ "embed"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/cksmvfs"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
)

//go:embed testdata/cksm.db
var cksmDB string

func Test_fileformat(t *testing.T) {
	readervfs.Create("test.db", ioutil.NewSizeReaderAt(strings.NewReader(cksmDB)))
	vfs.Register("rcksm", cksmvfs.Wrap(vfs.Find("reader")))

	db, err := driver.Open("file:test.db?vfs=rcksm")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled bool
	err = db.QueryRow(`PRAGMA checksum_verification`).Scan(&enabled)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("want true")
	}

	db.SetMaxIdleConns(0) // Clears the page cache.

	_, err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
}

//go:embed testdata/test.db
var testDB []byte

func Test_enable(t *testing.T) {
	memdb.Create("nockpt.db", testDB)
	vfs.Register("mcksm", cksmvfs.Wrap(vfs.Find("memdb")))

	db, err := driver.Open("file:/nockpt.db?vfs=mcksm",
		func(db *sqlite3.Conn) error {
			return cksmvfs.EnableChecksums(db, "")
		})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled bool
	err = db.QueryRow(`PRAGMA checksum_verification`).Scan(&enabled)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("want true")
	}

	db.SetMaxIdleConns(0) // Clears the page cache.

	_, err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_new(t *testing.T) {
	if !vfs.SupportsFileLocking {
		t.Skip("skipping without locks")
	}

	name := "file:" +
		filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?vfs=cksmvfs&_pragma=journal_mode(wal)"

	db, err := driver.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled bool
	err = db.QueryRow(`PRAGMA checksum_verification`).Scan(&enabled)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("want true")
	}

	var size int
	err = db.QueryRow(`PRAGMA page_size=1024`).Scan(&size)
	if err != nil {
		t.Fatal(err)
	}
	if size != 4096 {
		t.Errorf("got %d, want 4096", size)
	}

	_, err = db.Exec(`CREATE TABLE users (id INT, name VARCHAR(10))`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxIdleConns(0) // Clears the page cache.

	_, err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
}
