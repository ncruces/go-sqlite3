package readervfs_test

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "embed"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
	"github.com/psanford/httpreadat"
)

//go:embed testdata/test.db
var testDB string

func Example_http() {
	readervfs.Create("demo.db", httpreadat.New("https://sanford.io/demo.db"))
	defer readervfs.Delete("demo.db")

	db, err := sql.Open("sqlite3", "file:demo.db?vfs=reader")
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

func Example_embed() {
	readervfs.Create("test.db", ioutil.NewSizeReaderAt(strings.NewReader(testDB)))
	defer readervfs.Delete("test.db")

	db, err := sql.Open("sqlite3", "file:test.db?vfs=reader")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, name FROM users`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s\n", id, name)
	}
	// Output:
	// 0 go
	// 1 zig
	// 2 whatever
}
