package statement_test

import (
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/statement"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(statement.Register)
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
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

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
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
