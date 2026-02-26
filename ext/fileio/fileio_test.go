package fileio_test

import (
	"bytes"
	"database/sql"
	"io/fs"
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fileio"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_lsmode(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, fileio.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	s, err := os.Stat(d)
	if err != nil {
		t.Fatal(err)
	}

	var mode string
	err = db.QueryRow(`SELECT lsmode(?)`, s.Mode()).Scan(&mode)
	if err != nil {
		t.Fatal(err)
	}

	if len(mode) != 10 || mode[0] != 'd' {
		t.Errorf("got %s", mode)
	} else {
		t.Logf("got %s", mode)
	}
}

func Test_readfile(t *testing.T) {
	t.Parallel()

	for _, fsys := range []fs.FS{nil, os.DirFS(".")} {
		t.Run("", func(t *testing.T) {
			dsn := memdb.TestDB(t)

			db, err := driver.Open(dsn, func(c *sqlite3.Conn) error {
				fileio.RegisterFS(c, fsys)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			rows, err := db.Query(`SELECT readfile('fileio_test.go')`)
			if err != nil {
				t.Fatal(err)
			}

			if rows.Next() {
				var data sql.RawBytes
				rows.Scan(&data)

				if !bytes.HasPrefix(data, []byte("package fileio_test")) {
					t.Errorf("got %s", data[:min(64, len(data))])
				}
			}
		})
	}
}
