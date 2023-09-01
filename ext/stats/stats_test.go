package stats

import (
	"math"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestRegister_variance(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	Register(db)

	err = db.Exec(`CREATE TABLE IF NOT EXISTS data (x)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO data (x) VALUES (4), (7.0), ('13'), (NULL), (16)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT
			sum(x), avg(x),
			var_samp(x), var_pop(x),
			stddev_samp(x), stddev_pop(x)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		if got := stmt.ColumnFloat(0); got != 40 {
			t.Errorf("got %v, want 40", got)
		}
		if got := stmt.ColumnFloat(1); got != 10 {
			t.Errorf("got %v, want 10", got)
		}
		if got := stmt.ColumnFloat(2); got != 30 {
			t.Errorf("got %v, want 30", got)
		}
		if got := stmt.ColumnFloat(3); got != 22.5 {
			t.Errorf("got %v, want 22.5", got)
		}
		if got := stmt.ColumnFloat(4); got != math.Sqrt(30) {
			t.Errorf("got %v, want √30", got)
		}
		if got := stmt.ColumnFloat(5); got != math.Sqrt(22.5) {
			t.Errorf("got %v, want √22.5", got)
		}
	}

	{
		stmt, _, err := db.Prepare(`SELECT var_samp(x) OVER (ROWS 1 PRECEDING) FROM data`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()

		want := [...]float64{0, 4.5, 18, 0, 0}
		for i := 0; stmt.Step(); i++ {
			if got := stmt.ColumnFloat(0); got != want[i] {
				t.Errorf("got %v, want %v", got, want[i])
			}
			if got := stmt.ColumnType(0); (got == sqlite3.FLOAT) != (want[i] != 0) {
				t.Errorf("got %v, want %v", got, want[i])
			}
		}
	}
}

func TestRegister_covariance(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	Register(db)

	err = db.Exec(`CREATE TABLE IF NOT EXISTS data (x, y)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO data (x, y) VALUES (3, 70), (5, 80), (2, 60), (7, 90), (4, 75)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT
		corr(x, y), covar_samp(x, y), covar_pop(x, y) FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		if got := stmt.ColumnFloat(0); got != 0.9881049293224639 {
			t.Errorf("got %v, want 0.9881049293224639", got)
		}
		if got := stmt.ColumnFloat(1); got != 21.25 {
			t.Errorf("got %v, want 21.25", got)
		}
		if got := stmt.ColumnFloat(2); got != 17 {
			t.Errorf("got %v, want 17", got)
		}
	}

	{
		stmt, _, err := db.Prepare(`SELECT covar_samp(x, y) OVER (ROWS 1 PRECEDING) FROM data`)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()

		want := [...]float64{0, 10, 30, 75, 22.5}
		for i := 0; stmt.Step(); i++ {
			if got := stmt.ColumnFloat(0); got != want[i] {
				t.Errorf("got %v, want %v", got, want[i])
			}
			if got := stmt.ColumnType(0); (got == sqlite3.FLOAT) != (want[i] != 0) {
				t.Errorf("got %v, want %v", got, want[i])
			}
		}
	}
}
