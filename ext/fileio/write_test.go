package fileio

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_writefile(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, Register)
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
		err := rows.Scan(&name, &mode, &mtime, &data)
		if err != nil {
			t.Fatal(err)
		}
		if mode.IsDir() && !mtime.Equal(twosday) {
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
