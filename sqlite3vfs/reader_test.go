package sqlite3vfs_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "embed"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
	"github.com/psanford/httpreadat"
)

func ExampleReaderVFS() {
	sqlite3vfs.Register("httpvfs", sqlite3vfs.ReaderVFS{
		"demo.db": httpreadat.New("https://www.sanford.io/demo.db"),
	})

	db, err := sql.Open("sqlite3", "file:demo.db?vfs=httpvfs&mode=ro")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	magname := map[int]string{
		3: "thousand",
		6: "million",
		9: "billion",
	}
	rows, err := db.Query(`
		SELECT period, data_value, magntude, units FROM csv
			WHERE period > '2010'
			LIMIT 10`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var period, units string
		var value int64
		var mag int
		err = rows.Scan(&period, &value, &mag, &units)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d %s %s\n", period, value, magname[mag], units)
	}
	// Output:
	// 2010.03: 17463 million Dollars
	// 2010.06: 17260 million Dollars
	// 2010.09: 15419 million Dollars
	// 2010.12: 17088 million Dollars
	// 2011.03: 18516 million Dollars
	// 2011.06: 18835 million Dollars
	// 2011.09: 16390 million Dollars
	// 2011.12: 18748 million Dollars
	// 2012.03: 18477 million Dollars
	// 2012.06: 18270 million Dollars
}

//go:embed testdata/test.db
var testDB string

func TestReaderVFS_Open(t *testing.T) {
	sqlite3vfs.Register("reader", sqlite3vfs.ReaderVFS{
		"test.db": sqlite3vfs.NewSizeReaderAt(strings.NewReader(testDB)),
	})

	_, err := sqlite3.Open("file:demo.db?vfs=reader&mode=ro")
	if err == nil {
		t.Error("want error")
	}

	db, err := sqlite3.Open("file:test.db?vfs=reader&mode=ro")
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT id, name FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	row := 0
	ids := []int{0, 1, 2}
	names := []string{"go", "zig", "whatever"}
	for ; stmt.Step(); row++ {
		id := stmt.ColumnInt(0)
		name := stmt.ColumnText(1)

		if id != ids[row] {
			t.Errorf("got %d, want %d", id, ids[row])
		}
		if name != names[row] {
			t.Errorf("got %q, want %q", name, names[row])
		}
	}
	if row != 3 {
		t.Errorf("got %d, want %d", row, len(ids))
	}

	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewSizeReaderAt(t *testing.T) {
	n, err := sqlite3vfs.NewSizeReaderAt(strings.NewReader("abc")).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("got %d", n)
	}

	f, err := os.Create(filepath.Join(t.TempDir(), "abc.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	n, err = sqlite3vfs.NewSizeReaderAt(f).Size()
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("got %d", n)
	}
}
