package bench_test

import (
	"strconv"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func init() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		panic(err)
	}
	err = db.Close()
	if err != nil {
		panic(err)
	}
}

func BenchmarkWasm(b *testing.B) {
	for n := 0; n < b.N; n++ {
		wasmTest()
	}
}

func wasmTest() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY NOT NULL, name VARCHAR, age INTEGER, rating REAL)`)
	if err != nil {
		panic(err)
	}

	func() {
		const N = 1_000_000

		db.Exec(`BEGIN`)
		defer db.Exec(`END`)

		stmt, _, err := db.Prepare(`INSERT INTO users (id, name, age, rating) VALUES (?, ?, ?, ?)`)
		if err != nil {
			panic(err)
		}
		defer stmt.Close()

		for i := 0; i < N; i++ {
			id := i + 1
			name := "user " + strconv.Itoa(id)
			age := 33 + id
			rating := 0.13 * float64(id)
			stmt.BindInt(1, id)
			stmt.BindText(2, name)
			stmt.BindInt64(3, int64(age))
			stmt.BindFloat(4, rating)
		}
	}()

	func() {
		stmt, _, err := db.Prepare(`SELECT id, name, age, rating FROM users ORDER BY id`)
		if err != nil {
			panic(err)
		}
		defer stmt.Close()

		for {
			if row, err := stmt.Step(); err != nil {
				panic(err)
			} else if !row {
				break
			}
			id := stmt.ColumnInt(0)
			name := stmt.ColumnText(1)
			age := stmt.ColumnInt64(2)
			rating := stmt.ColumnFloat(3)
			if id < 1 || len(name) < 5 || age < 33 || rating < 0.13 {
				panic("wrong row values")
			}
		}
	}()
}
