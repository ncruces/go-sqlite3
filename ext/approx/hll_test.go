package approx_test

import (
	"math"
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/approx"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(approx.Register)
	os.Exit(m.Run())
}

func TestRegister_count_distinct(t *testing.T) {
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

	// Empty table should return 0 (matching COUNT(DISTINCT x)).
	stmt, _, err := db.Prepare(`SELECT approx_count_distinct(x), count(DISTINCT x) FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else if got, want := stmt.ColumnInt64(0), stmt.ColumnInt64(1); got != want {
		t.Errorf("empty table: got %v, want %v", got, want)
	}
	stmt.Close()

	// Mixed types: NULLs ignored, 1 == 1.0, '1' != 1, X'31' != '1', large ints.
	err = db.Exec(`
		INSERT INTO data (x) VALUES
			(NULL), (NULL),
			(0), (-0.0),
			(1), (1.0), ('1'), (X'31'),
			(2.5), (2.5),
			(9007199254740992), (9007199254740993), (9007199254740992.0)
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`SELECT approx_count_distinct(x), count(DISTINCT x) FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else if got, want := stmt.ColumnInt64(0), stmt.ColumnInt64(1); got != want {
		t.Errorf("mixed types: got %v, want %v", got, want)
	}
	stmt.Close()
}

func TestRegister_window(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE data (x);
		INSERT INTO data (x) VALUES
			(NULL),
			(1),
			(1.0),
			(2),
			('2'),
			(X'32'),
			(NULL),
			(3),
			(3),
			(4);
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT
			approx_count_distinct(x) OVER (ROWS BETWEEN 2 PRECEDING AND 1 FOLLOWING)
		FROM data
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	// Windows (2 preceding to 1 following):
	// i=0: [NULL, 1] -> {1} -> 1
	// i=1: [NULL, 1, 1.0] -> {1} -> 1
	// i=2: [NULL, 1, 1.0, 2] -> {1, 2} -> 2
	// i=3: [1, 1.0, 2, '2'] -> {1, 2, '2'} -> 3
	// i=4: [1.0, 2, '2', X'32'] -> {1, 2, '2', X'32'} -> 4
	// i=5: [2, '2', X'32', NULL] -> {2, '2', X'32'} -> 3
	// i=6: ['2', X'32', NULL, 3] -> {'2', X'32', 3} -> 3
	// i=7: [X'32', NULL, 3, 3] -> {X'32', 3} -> 2
	// i=8: [NULL, 3, 3, 4] -> {3, 4} -> 2
	// i=9: [3, 3, 4] -> {3, 4} -> 2
	want := [...]int64{1, 1, 2, 3, 4, 3, 3, 2, 2, 2}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnInt64(0); got != want[i] {
			t.Errorf("row %d: got %v, want %v", i, got, want[i])
		}
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestRegister_large(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Standard aggregate exceeding maxWindow (1e4) degrades gracefully and transitions HLL from sparse to dense.
	const n = 50000
	stmt, _, err := db.Prepare(`
		SELECT
			approx_count_distinct(value % ?),
			approx_count_distinct(CAST(value % ? AS TEXT))
		FROM generate_series(1, ?)
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	stmt.BindInt(1, n)
	stmt.BindInt(2, n)
	stmt.BindInt(3, 2*n)

	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	for c := range 2 {
		got := stmt.ColumnInt64(c)
		relErr := math.Abs(float64(got-n)) / n
		if relErr > 0.03 {
			t.Errorf("col %d: got %d, want ~%d (relErr %.2f%%)", c, got, n, relErr*100)
		}
	}

	// Sliding window with medium cardinality per window.
	stmt2, _, err := db.Prepare(`
		SELECT approx_count_distinct(value % 50) OVER (ROWS 100 PRECEDING)
		FROM generate_series(1, 500)
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt2.Close()

	for i := 1; stmt2.Step(); i++ {
		got := stmt2.ColumnInt64(0)
		want := int64(min(i, 50))
		if math.Abs(float64(got-want)) > 2 {
			t.Errorf("row %d: got %d, want ~%d", i, got, want)
		}
	}
	if err := stmt2.Err(); err != nil {
		t.Fatal(err)
	}
}

