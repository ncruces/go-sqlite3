//go:build !js

package array_test

import (
	"fmt"
	"log"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/array"
)

func Example_driver() {
	db, err := driver.Open("file:/test.db?vfs=memdb", array.Register)
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

func Example() {
	sqlite3.AutoExtension(array.Register)

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`
		SELECT name
		FROM pragma_function_list
		WHERE name like 'geopoly%' AND narg IN array(?)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.BindPointer(1, [...]int{2, 3, 4})
	if err != nil {
		log.Fatal(err)
	}

	for stmt.Step() {
		fmt.Printf("%s\n", stmt.ColumnText(0))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}
	// Unordered output:
	// geopoly_regular
	// geopoly_overlap
	// geopoly_contains_point
	// geopoly_within
}
