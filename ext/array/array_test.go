package array_test

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"testing"

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

func Test_cursor_Column(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		array.Register(c)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT rowid, value FROM array(?)`,
		sqlite3.Pointer(&[...]any{nil, true, 1, uint(2), math.Pi, "text", []byte{1, 2, 3}}))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	want := []string{"nil", "int64", "int64", "int64", "float64", "string", "[]uint8"}

	for rows.Next() {
		var id, val any
		err := rows.Scan(&id, &val)
		if err != nil {
			t.Fatal(err)
		}
		if want := want[0]; val == nil {
			if want != "nil" {
				t.Errorf("got nil, want %s", want)
			}
		} else if got := reflect.TypeOf(val).String(); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
		want = want[1:]
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func Test_array_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	array.Register(db)

	err = db.Exec(`SELECT * FROM array()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`SELECT * FROM array(?)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}
