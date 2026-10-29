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

func TestRegister_hll(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE data (x, y)`)
	if err != nil {
		t.Fatal(err)
	}

	// Empty table.
	stmt, _, err := db.Prepare(`
		SELECT
			approx_count_distinct(x),
			approx_count_distinct_estimate(approx_count_distinct_accumulate(x)),
			approx_count_distinct_combine(x),
			approx_count_distinct_estimate(NULL)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	if got := stmt.ColumnInt64(0); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := stmt.ColumnInt64(1); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := stmt.ColumnType(2); got != sqlite3.NULL {
		t.Errorf("got %v, want NULL", got)
	}
	if got := stmt.ColumnType(3); got != sqlite3.NULL {
		t.Errorf("got %v, want NULL", got)
	}
	stmt.Close()

	// Mixed types: NULLs ignored, 1 == 1.0, '1' != 1, X'31' != '1', large ints, tuples.
	err = db.Exec(`
		INSERT INTO data (x, y) VALUES
			(NULL, 1),
			(0, NULL),
			(-0.0, 1),
			(1, 2),
			(1.0, 2.0),
			('1', 2),
			(X'31', 2),
			(2.5, 1),
			(2.5, 1),
			(9007199254740992, 1),
			(9007199254740993, 1),
			(9007199254740992.0, 1)
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`
		SELECT
			count(DISTINCT x),
			approx_count_distinct(x),
			approx_count_distinct(x, y),
			approx_count_distinct()
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	if got, want := stmt.ColumnInt64(1), stmt.ColumnInt64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := stmt.ColumnInt64(2); got != 7 {
		t.Errorf("got %v, want 7", got)
	}
	if got := stmt.ColumnInt64(3); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	stmt.Close()

	// Sliding window.
	stmt, _, err = db.Prepare(`
		SELECT
			approx_count_distinct(x) OVER w,
			approx_count_distinct_estimate(approx_count_distinct_accumulate(x) OVER w)
		FROM data
		WINDOW w AS (ROWS BETWEEN 2 PRECEDING AND 1 FOLLOWING)`)
	if err != nil {
		t.Fatal(err)
	}
	want := [...]int64{1, 1, 2, 2, 3, 3, 4, 3, 3, 3, 3, 2}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnInt64(0); got != want[i] {
			t.Errorf("row %d: got %v, want %v", i, got, want[i])
		}
		if got := stmt.ColumnInt64(1); got != want[i] {
			t.Errorf("row %d: got %v, want %v", i, got, want[i])
		}
	}
	stmt.Close()
}

func TestRegister_large(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE sketches (grp, sketch);
		INSERT INTO sketches VALUES (0, NULL);
		INSERT INTO sketches SELECT 1, approx_count_distinct_accumulate(NULL);
		INSERT INTO sketches
		SELECT value % 4 + 2, approx_count_distinct_accumulate(value % 25000)
		FROM generate_series(1, 50000)
		GROUP BY value % 4;
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT
			approx_count_distinct_estimate(approx_count_distinct_combine(sketch) OVER ()),
			approx_count_distinct_estimate(approx_count_distinct_combine(sketch) OVER (ORDER BY grp ROWS 2 PRECEDING))
		FROM sketches`)
	if err != nil {
		t.Fatal(err)
	}
	want := [...]float64{0, 0, 6250, 12500, 18750, 18750}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnFloat(0); math.Abs(got-25000)/25000 > 0.05 {
			t.Errorf("row %d: got %.0f, want ~25000", i, got)
		}
		if i == 0 && stmt.ColumnType(1) != sqlite3.NULL {
			t.Errorf("row 0: got %v, want NULL", stmt.ColumnType(1))
		}
		if got := stmt.ColumnFloat(1); math.Abs(got-want[i]) > 0.05*want[i] {
			t.Errorf("row %d: got %.0f, want ~%.0f", i, got, want[i])
		}
	}
	stmt.Close()
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []string{
		`SELECT approx_count_distinct_estimate('not a blob')`,
		`SELECT approx_count_distinct_estimate(X'00')`,
		`SELECT approx_count_distinct_combine('not a blob')`,
		`SELECT approx_count_distinct_combine(X'00')`,
		`SELECT approx_count_distinct_combine(column1) FROM (VALUES (X'118A7F'), (X'118B7F'))`,
		`SELECT approx_count_distinct_combine(column1) OVER (ROWS 1 PRECEDING) FROM (VALUES (X'118A7F'), (X'118B7F'))`,
	}
	for _, sql := range tests {
		if err := db.Exec(sql); err == nil {
			t.Errorf("expected error for %q", sql)
		}
	}
}

func Benchmark_hll(b *testing.B) {
	db, err := sqlite3.OpenContext(b.Context(), ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE data (grp10 INT, grp2 INT, x INT);
		INSERT INTO data SELECT value / 10, value / 2, value FROM generate_series(0, 49999);
		CREATE INDEX idx_grp10 ON data (grp10, x);
		CREATE INDEX idx_grp2 ON data (grp2, x);
	`)
	if err != nil {
		b.Fatal(err)
	}

	queries := []struct {
		name, sql string
	}{
		{"agg_5k", `SELECT approx_count_distinct(x) FROM data WHERE rowid <= 5000`},
		{"agg_50k", `SELECT approx_count_distinct(x) FROM data`},
		{"group_2k_x2", `SELECT approx_count_distinct(x) FROM data WHERE grp2 < 2000 GROUP BY grp2`},
		{"group_1k_x10", `SELECT approx_count_distinct(x) FROM data WHERE grp10 < 1000 GROUP BY grp10`},
		{"win_20_explicit", `SELECT approx_count_distinct(x) OVER (ROWS 20 PRECEDING) FROM data WHERE rowid <= 2000`},
		{"win_250_sparse", `SELECT approx_count_distinct(x) OVER (ROWS 250 PRECEDING) FROM data WHERE rowid <= 2000`},
		{"win_1000_dense", `SELECT approx_count_distinct(x) OVER (ROWS 1000 PRECEDING) FROM data WHERE rowid <= 2000`},
	}

	for _, q := range queries {
		b.Run(q.name, func(b *testing.B) {
			b.ReportAllocs()
			stmt, _, err := db.Prepare(q.sql)
			if err != nil {
				b.Fatal(err)
			}
			defer stmt.Close()

			for b.Loop() {
				for stmt.Step() {
				}
				if err := stmt.Reset(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
