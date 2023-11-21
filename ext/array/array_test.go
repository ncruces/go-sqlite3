package array_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/array"
)

func Example() {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		array.Register(c)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT name
		FROM pragma_function_list
		WHERE name like 'geopoly%' AND narg IN array(?)`,
		sqlite3.Pointer([]int{2, 3, 4}))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", name)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	// Unordered output:
	// geopoly_regular
	// geopoly_overlap
	// geopoly_contains_point
	// geopoly_within
}
