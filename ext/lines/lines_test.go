package lines_test

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/lines"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example() {
	db, err := driver.Open("file:/test.db?vfs=memdb", lines.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	res, err := http.Get("https://storage.googleapis.com/quickdraw_dataset/full/simplified/calendar.ndjson")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	rows, err := db.Query(`
		SELECT
			line ->> '$.countrycode' as countrycode,
			COUNT(*)
  		FROM lines_read(?)
  		GROUP BY 1
  		ORDER BY 2 DESC
  		LIMIT 5`,
		sqlite3.Pointer(res.Body))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var countrycode sql.RawBytes
	var count int
	for rows.Next() {
		err := rows.Scan(&countrycode, &count)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d\n", countrycode, count)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	// Expected output:
	// US: 141001
	// GB: 22560
	// CA: 11759
	// RU: 9250
	// DE: 8748
}

func Test_lines(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, lines.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	const data = "line 1\nline 2\r\nline 3\n"

	rows, err := db.Query(`SELECT rowid, line FROM lines(?)`, data)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var line string
		err := rows.Scan(&id, &line)
		if err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf("line %d", id); line != want {
			t.Errorf("got %q, want %q", line, want)
		}
	}
}

func Test_lines_error(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, lines.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`SELECT rowid, line FROM lines(?)`, nil)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	_, err = db.Exec(`SELECT rowid, line FROM lines_read(?)`, "xpto")
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}

func Test_lines_read(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, lines.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	const data = "line 1\nline 2\r\nline 3\n"

	rows, err := db.Query(`SELECT rowid, line FROM lines_read(?)`,
		sqlite3.Pointer(strings.NewReader(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var line string
		err := rows.Scan(&id, &line)
		if err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf("line %d", id); line != want {
			t.Errorf("got %q, want %q", line, want)
		}
	}
}

func Test_lines_test(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, lines.Register)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT rowid, line FROM lines_read(?)`, "lines_test.go")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var line string
		err := rows.Scan(&id, &line)
		if err != nil {
			t.Fatal(err)
		}
	}
}
