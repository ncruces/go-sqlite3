package fileio_test

import (
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/fileio"
)

func Test_lsmode(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		fileio.Register(c)
		return nil
	})
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
