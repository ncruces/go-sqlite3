package fileio_test

import (
	"io/fs"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/fileio"
)

func Test_fsdir(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		fileio.Register(c)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT name, mode, mtime FROM fsdir('.')`)
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var name string
		var mode fs.FileMode
		var mtime time.Time
		err := rows.Scan(&name, &mode, sqlite3.TimeFormatUnixFrac.Scanner(&mtime))
		if err != nil {
			t.Fatal(err)
		}
		if mode.Perm() == 0 {
			t.Errorf("mode %v", mode)
		}
		if mtime.Before(time.Unix(0, 0)) {
			t.Errorf("mtime %v", mtime)
		}
		t.Log(name)
	}
}
