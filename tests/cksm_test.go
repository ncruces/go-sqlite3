package tests

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
)

//go:embed testdata/cksm.db
var cksmDB string

func Test_fileformat(t *testing.T) {
	t.Parallel()

	readervfs.Create("test.db", ioutil.NewSizeReaderAt(strings.NewReader(cksmDB)))

	db, err := driver.Open("file:test.db?vfs=reader")
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

func Test_enable(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(memdb.TestDB(t),
		func(db *sqlite3.Conn) error {
			return db.EnableChecksums("main")
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
