package stats_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/stats"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_quantile(t *testing.T) {
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
			median(x),
			quantile_disc(x, 0.5),
			quantile_cont(x, 0.25)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnFloat(0); got != 10 {
			t.Errorf("got %v, want 10", got)
		}
		if got := stmt.ColumnFloat(1); got != 7 {
			t.Errorf("got %v, want 7", got)
		}
		if got := stmt.ColumnFloat(2); got != 6.25 {
			t.Errorf("got %v, want 6.25", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`
		SELECT
			median(x),
			quantile_disc(x, 0.5),
			quantile_cont(x, 0.25)
		FROM data
		WHERE x < 5`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnFloat(0); got != 4 {
			t.Errorf("got %v, want 4", got)
		}
		if got := stmt.ColumnFloat(1); got != 4 {
			t.Errorf("got %v, want 4", got)
		}
		if got := stmt.ColumnFloat(2); got != 4 {
			t.Errorf("got %v, want 4", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`
		SELECT
			median(x),
			quantile_disc(x, 0.5),
			quantile_cont(x, 0.25)
		FROM data
		WHERE x < 0`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.NULL {
			t.Error("want NULL")
		}
		if got := stmt.ColumnType(1); got != sqlite3.NULL {
			t.Error("want NULL")
		}
		if got := stmt.ColumnType(2); got != sqlite3.NULL {
			t.Error("want NULL")
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`
		SELECT
			quantile_disc(x, -2),
			quantile_cont(x, +2)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		t.Fatal("want error")
	}
	stmt.Close()
}
