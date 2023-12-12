package fileio

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func Test_writefile(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		Register(c)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	file := filepath.Join(dir, "test.txt")
	nest := filepath.Join(dir, "tmp", "test.txt")
	sock := filepath.Join(dir, "sock")
	twosday := time.Date(2022, 2, 22, 22, 22, 22, 0, time.UTC)

	_, err = db.Exec(`SELECT writefile(?, 'Hello world!')`, file)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`SELECT writefile(?, ?, ?)`, link, "test.txt", fs.ModeSymlink)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`SELECT writefile(?, ?, ?, ?)`, dir, nil, 0040700, twosday.Unix())
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`SELECT * FROM fsdir('.', ?)`, dir)
	if err != nil {
		t.Fatal(err)
	}

	for rows.Next() {
		var name string
		var mode fs.FileMode
		var mtime time.Time
		var data sql.NullString
		err := rows.Scan(&name, &mode, sqlite3.TimeFormatUnixFrac.Scanner(&mtime), &data)
		if err != nil {
			t.Fatal(err)
		}
		if mode.IsDir() && mtime != twosday {
			t.Errorf("got: %v", mtime)
		}
		if mode.IsRegular() && data.String != "Hello world!" {
			t.Errorf("got: %v", data)
		}
		if mode&fs.ModeSymlink != 0 && data.String != "test.txt" {
			t.Errorf("got: %v", data)
		}
	}

	_, err = db.Exec(`SELECT writefile(?, 'Hello world!')`, nest)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`SELECT writefile(?, ?, ?)`, sock, nil, fs.ModeSocket)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	_, err = db.Exec(`SELECT writefile()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}

func Test_fixMode(t *testing.T) {
	tests := []struct {
		mode fs.FileMode
		want fs.FileMode
	}{
		{0010754, 0754 | fs.ModeNamedPipe},
		{0020754, 0754 | fs.ModeCharDevice | fs.ModeDevice},
		{0040754, 0754 | fs.ModeDir},
		{0060754, 0754 | fs.ModeDevice},
		{0100754, 0754},
		{0120754, 0754 | fs.ModeSymlink},
		{0140754, 0754 | fs.ModeSocket},
		{0170754, 0754 | fs.ModeIrregular},
	}
	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			if got := fixMode(tt.mode); got != tt.want {
				t.Errorf("fixMode() = %o, want %o", got, tt.want)
			}
		})
	}
}
