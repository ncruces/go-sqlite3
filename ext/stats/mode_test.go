package stats_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_mode(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT mode(column1) FROM (VALUES (NULL), (1), (NULL), (2), (NULL), (3), (3))`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnInt(0); got != 3 {
			t.Errorf("got %v, want 3", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`SELECT mode(column1) FROM (VALUES (1), (1), (2), (2), (3))`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnInt(0); got != 1 {
			t.Errorf("got %v, want 1", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`SELECT mode(column1) FROM (VALUES (0.5), (1), (2.5), (2), (2.5))`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnFloat(0); got != 2.5 {
			t.Errorf("got %v, want 2.5", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`SELECT mode(column1) FROM (VALUES ('red'), ('green'), ('blue'), ('red'))`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnText(0); got != "red" {
			t.Errorf("got %q, want red", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`SELECT mode(column1) FROM (VALUES (X'cafebabe'), ('green'), ('blue'), (X'cafebabe'))`)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Step() {
		if got := stmt.ColumnText(0); got != "\xca\xfe\xba\xbe" {
			t.Errorf("got %q, want cafebabe", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`
		SELECT mode(column1) OVER (ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING)
		FROM (VALUES (1), (1), (2.5), ('blue'), (X'cafebabe'), (1), (1))
	`)
	if err != nil {
		t.Fatal(err)
	}
	for stmt.Step() {
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`SELECT mode(column1) FROM (VALUES (?), (?), (?), (?), (?))`)
	if err != nil {
		t.Fatal(err)
	}
	stmt.BindInt(1, 1)
	stmt.BindInt(2, 1)
	stmt.BindInt(3, 2)
	stmt.BindFloat(4, 2)
	stmt.BindFloat(5, 2)
	if stmt.Step() {
		if got := stmt.ColumnInt(0); got != 2 {
			t.Errorf("got %v, want 2", got)
		}
	}
	stmt.Close()
}
