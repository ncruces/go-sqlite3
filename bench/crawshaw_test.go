package bench_test

import (
	"strconv"
	"testing"

	"crawshaw.io/sqlite"
)

func BenchmarkCrawshaw(b *testing.B) {
	for n := 0; n < b.N; n++ {
		crawshawTest()
	}
}

func crawshawTest() {
	db, err := sqlite.OpenConn(":memory:", 0)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	exec := func(sql string) error {
		stmt, _, err := db.PrepareTransient(sql)
		if err != nil {
			return err
		}
		_, err = stmt.Step()
		if err != nil {
			return err
		}
		return stmt.Finalize()
	}

	err = exec(`CREATE TABLE users (id INTEGER PRIMARY KEY NOT NULL, name VARCHAR, age INTEGER, rating REAL)`)
	if err != nil {
		panic(err)
	}

	func() {
		const N = 1_000_000

		exec(`BEGIN`)
		defer exec(`END`)

		stmt, _, err := db.PrepareTransient(`INSERT INTO users (id, name, age, rating) VALUES (?, ?, ?, ?)`)
		if err != nil {
			panic(err)
		}
		defer stmt.Finalize()

		for i := 0; i < N; i++ {
			id := i + 1
			name := "user " + strconv.Itoa(id)
			age := 33 + id
			rating := 0.13 * float64(id)
			stmt.BindInt64(1, int64(id))
			stmt.BindText(2, name)
			stmt.BindInt64(3, int64(age))
			stmt.BindFloat(4, rating)
		}
	}()

	func() {
		stmt, _, err := db.PrepareTransient(`SELECT id, name, age, rating FROM users ORDER BY id`)
		if err != nil {
			panic(err)
		}
		defer stmt.Finalize()

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
