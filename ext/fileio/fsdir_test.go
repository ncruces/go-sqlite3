package fileio_test

import (
	"bytes"
	"database/sql"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/fileio"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func Test_fsdir(t *testing.T) {
	t.Parallel()

	for _, fsys := range []fs.FS{nil, os.DirFS(".")} {
		t.Run("", func(t *testing.T) {
			db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
				fileio.RegisterFS(c, fsys)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			rows, err := db.Query(`SELECT * FROM fsdir('.', '.')`)
			if err != nil {
				t.Fatal(err)
			}

			for rows.Next() {
				var name string
				var mode fs.FileMode
				var mtime time.Time
				var data sql.RawBytes
				err := rows.Scan(&name, &mode, sqlite3.TimeFormatUnixFrac.Scanner(&mtime), &data)
				if err != nil {
					t.Fatal(err)
				}
				if mode.Perm() == 0 {
					t.Errorf("got: %v", mode)
				}
				if mtime.Before(time.Unix(0, 0)) {
					t.Errorf("got: %v", mtime)
				}
				if name == "fsdir_test.go" {
					if !bytes.HasPrefix(data, []byte("package fileio_test")) {
						t.Errorf("got: %s", data[:min(64, len(data))])
					}
				}
			}
		})
	}
}

func Test_fsdir_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	fileio.Register(db)

	err = db.Exec(`SELECT name FROM fsdir()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}
