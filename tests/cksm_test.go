package tests

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/internal/testutil"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
)

//go:embed testdata/cksm.db
var cksmDB string

func Test_fileformat(t *testing.T) {
	t.Parallel()

	readervfs.Create("test.db", ioutil.NewSizeReaderAt(strings.NewReader(cksmDB)))

	ctx := testutil.Context(t)
	db, err := driver.Open("file:test.db?vfs=reader")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled bool
	err = db.QueryRowContext(ctx, `PRAGMA checksum_verification`).Scan(&enabled)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("want true")
	}

	db.SetMaxIdleConns(0) // Clears the page cache.

	_, err = db.ExecContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_enable(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db, err := driver.Open(memdb.TestDB(t),
		func(db *sqlite3.Conn) error {
			return db.EnableChecksums("main")
		})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled bool
	err = db.QueryRowContext(ctx, `PRAGMA checksum_verification`).Scan(&enabled)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("want true")
	}

	db.SetMaxIdleConns(0) // Clears the page cache.

	_, err = db.ExecContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
}
