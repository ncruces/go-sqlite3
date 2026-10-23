package stats_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_bitwise(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE data (x)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT bit_and(x), bit_or(x), bit_xor(x) FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else {
		for i := range 3 {
			if got := stmt.ColumnType(i); got != sqlite3.NULL {
				t.Errorf("got %v, want NULL", got)
			}
		}
	}
	stmt.Close()

	err = db.Exec(`INSERT INTO data (x) VALUES (3), (6.0), ('12'), (10), (NULL), ('abc')`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`
		SELECT
			bit_and(x),
			bit_and(x | 1),
			bit_or(x),
			bit_xor(x)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else {
		if got := stmt.ColumnInt64(0); got != 0 {
			t.Errorf("got %v, want 0", got)
		}
		if got := stmt.ColumnInt64(1); got != 1 {
			t.Errorf("got %v, want 1", got)
		}
		if got := stmt.ColumnInt64(2); got != 15 {
			t.Errorf("got %v, want 15", got)
		}
		if got := stmt.ColumnInt64(3); got != 3 {
			t.Errorf("got %v, want 3", got)
		}
	}
	stmt.Close()

	stmt, _, err = db.Prepare(`
		SELECT
			bit_and(x) OVER (ROWS 1 PRECEDING),
			bit_or(x)  OVER (ROWS 1 PRECEDING),
			bit_xor(x) OVER (ROWS 1 PRECEDING)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}

	wantAnd := [...]int64{3, 2, 4, 8, 10, 0}
	wantOr := [...]int64{3, 7, 14, 14, 10, 0}
	wantXor := [...]int64{3, 5, 10, 6, 10, 0}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnInt64(0); got != wantAnd[i] {
			t.Errorf("row %d: got %v, want %v", i, got, wantAnd[i])
		}
		if got := stmt.ColumnInt64(1); got != wantOr[i] {
			t.Errorf("row %d: got %v, want %v", i, got, wantOr[i])
		}
		if got := stmt.ColumnInt64(2); got != wantXor[i] {
			t.Errorf("row %d: got %v, want %v", i, got, wantXor[i])
		}
		wantType := sqlite3.INTEGER
		if i == 5 {
			wantType = sqlite3.NULL
		}
		for c := range 3 {
			if got := stmt.ColumnType(c); got != wantType {
				t.Errorf("row %d col %d: got %v, want %v", i, c, got, wantType)
			}
		}
	}
	stmt.Close()
}
