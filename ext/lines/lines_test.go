package lines_test

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/lines"
)

func Example() {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		lines.Register(c)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// https://storage.googleapis.com/quickdraw_dataset/full/simplified/calendar.ndjson
	f, err := os.Open("calendar.ndjson")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	rows, err := db.Query(`
		SELECT
			line ->> '$.countrycode' as countrycode,
			COUNT(*)
  		FROM lines_read(?)
  		GROUP BY 1
  		ORDER BY 2 DESC
  		LIMIT 5`,
		sqlite3.Pointer(f))
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
	// Sample output:
	// US: 141001
	// GB: 22560
	// CA: 11759
	// RU: 9250
	// DE: 8748
}

func Test_lines(t *testing.T) {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		lines.Register(c)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	const data = "line 1\nline 2\nline 3"

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
	}
}

func Test_lines_error(t *testing.T) {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		lines.Register(c)
		return nil
	})
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
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		lines.Register(c)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	const data = "line 1\nline 2\nline 3"

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
	}
}

func Test_lines_test(t *testing.T) {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		lines.Register(c)
		return nil
	})
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
