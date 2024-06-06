package stats_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/stats"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_boolean(t *testing.T) {
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

	err = db.Exec(`INSERT INTO data (x) VALUES (4), (7.0), (13), (NULL), (16), (3.14)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT
			every(x > 0),
			every(x > 10),
			some(x > 10),
			some(x > 20)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnBool(0); got != true {
			t.Errorf("got %v, want true", got)
		}
		if got := stmt.ColumnBool(1); got != false {
			t.Errorf("got %v, want false", got)
		}
		if got := stmt.ColumnBool(2); got != true {
			t.Errorf("got %v, want true", got)
		}
		if got := stmt.ColumnBool(3); got != false {
			t.Errorf("got %v, want false", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`SELECT every(x > 10) OVER (ROWS 1 PRECEDING) FROM data`)
	if err != nil {
		t.Fatal(err)
	}

	want := [...]bool{false, false, false, true, true, false}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnBool(0); got != want[i] {
			t.Errorf("got %v, want %v", got, want[i])
		}
		if got := stmt.ColumnType(0); got != sqlite3.INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
	}
	stmt.Close()
}
