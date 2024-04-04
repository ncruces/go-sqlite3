package stats_test

import (
	"math"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/stats"
)

func TestRegister_variance(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stats.Register(db)

	err = db.Exec(`CREATE TABLE data (x)`)
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

	stats.Register(db)

	err = db.Exec(`CREATE TABLE data (y, x)`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO data (y, x) VALUES (3, 70), (5, 80), (2, 60), (7, 90), (4, 75)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT
		corr(y, x), covar_samp(y, x), covar_pop(y, x),
		regr_avgy(y, x), regr_avgx(y, x),
		regr_syy(y, x), regr_sxx(y, x), regr_sxy(y, x),
		regr_slope(y, x), regr_intercept(y, x), regr_r2(y, x),
		regr_count(y, x), regr_json(y, x)
		FROM data`)
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
		if got := stmt.ColumnFloat(3); got != 4.2 {
			t.Errorf("got %v, want 4.2", got)
		}
		if got := stmt.ColumnFloat(4); got != 75 {
			t.Errorf("got %v, want 75", got)
		}
		if got := stmt.ColumnFloat(5); got != 14.8 {
			t.Errorf("got %v, want 14.8", got)
		}
		if got := stmt.ColumnFloat(6); got != 500 {
			t.Errorf("got %v, want 500", got)
		}
		if got := stmt.ColumnFloat(7); got != 85 {
			t.Errorf("got %v, want 85", got)
		}
		if got := stmt.ColumnFloat(8); got != 0.17 {
			t.Errorf("got %v, want 0.17", got)
		}
		if got := stmt.ColumnFloat(9); got != -8.55 {
			t.Errorf("got %v, want -8.55", got)
		}
		if got := stmt.ColumnFloat(10); got != 0.9763513513513513 {
			t.Errorf("got %v, want 0.9763513513513513", got)
		}
		if got := stmt.ColumnInt(11); got != 5 {
			t.Errorf("got %v, want 5", got)
		}
		var a map[string]float64
		if err := stmt.ColumnJSON(12, &a); err != nil {
			t.Error(err)
		} else if got := a["count"]; got != 5 {
			t.Errorf("got %v, want 5", got)
		}
	}

	{
		stmt, _, err := db.Prepare(`SELECT covar_samp(y, x) OVER (ROWS 1 PRECEDING) FROM data`)
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

func Benchmark_average(b *testing.B) {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT avg(value) FROM generate_series(0, ?)`)
	if err != nil {
		b.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.BindInt(1, b.N)
	if err != nil {
		b.Fatal(err)
	}

	if stmt.Step() {
		want := float64(b.N) / 2
		if got := stmt.ColumnFloat(0); got != want {
			b.Errorf("got %v, want %v", got, want)
		}
	}

	err = stmt.Err()
	if err != nil {
		b.Error(err)
	}
}

func Benchmark_variance(b *testing.B) {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	stats.Register(db)

	stmt, _, err := db.Prepare(`SELECT var_pop(value) FROM generate_series(0, ?)`)
	if err != nil {
		b.Fatal(err)
	}
	defer stmt.Close()

	err = stmt.BindInt(1, b.N)
	if err != nil {
		b.Fatal(err)
	}

	if stmt.Step() && b.N > 100 {
		want := float64(b.N*b.N) / 12
		if got := stmt.ColumnFloat(0); want > (got-want)*float64(b.N) {
			b.Errorf("got %v, want %v", got, want)
		}
	}

	err = stmt.Err()
	if err != nil {
		b.Error(err)
	}
}
