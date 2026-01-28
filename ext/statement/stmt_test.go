package statement_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/statement"
)

func Example() {
	sqlite3.AutoExtension(statement.Register)

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE VIRTUAL TABLE split_date USING statement((
			SELECT
				strftime('%Y', :date) AS year,
				strftime('%m', :date) AS month,
				strftime('%d', :date) AS day
		))`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT * FROM split_date('2022-02-22')`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		fmt.Printf("Twosday was %d-%d-%d", stmt.ColumnInt(0), stmt.ColumnInt(1), stmt.ColumnInt(2))
	}
	if err := stmt.Reset(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// Twosday was 2022-2-22
}

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(statement.Register)
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE VIRTUAL TABLE arguments USING statement((SELECT ? AS a, ? AS b, ? AS c))
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`
		SELECT * from arguments WHERE [2] = 'y' AND [3] = 'z'
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`
		CREATE VIRTUAL TABLE hypot USING statement((SELECT sqrt(:x * :x + :y * :y) AS hypotenuse))
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT x, y, * FROM hypot WHERE x = 3 AND y = 4
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else {
		x := stmt.ColumnInt(0)
		y := stmt.ColumnInt(1)
		hypot := stmt.ColumnInt(2)
		if x != 3 || y != 4 || hypot != 5 {
			t.Errorf("hypot(%d, %d) = %d", x, y, hypot)
		}
	}
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING statement()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING statement(SELECT 1, SELECT 2)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING statement((SELECT 1, SELECT 2))`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING statement((SELECT 1; SELECT 2))`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE split_date USING statement((CREATE TABLE x(val)))`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}
