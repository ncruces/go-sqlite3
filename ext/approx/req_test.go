package approx_test

import (
	"math"
	"slices"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_req(t *testing.T) {
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

	// Empty table.
	stmt, _, err := db.Prepare(`
		SELECT
			approx_percentile(x, 0.5),
			approx_percent_rank(x, 10),
			approx_percentile_estimate(approx_percentile_accumulate(x), 0.5),
			approx_percent_rank_estimate(approx_percentile_accumulate(x), 10),
			approx_percentile_combine(x),
			approx_percentile_estimate(NULL, 0.5),
			approx_percent_rank_estimate(NULL, 10)
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	for i := range 7 {
		if got := stmt.ColumnType(i); got != sqlite3.NULL {
			t.Errorf("col %d: got %v, want NULL", i, got)
		}
	}
	stmt.Close()

	err = db.Exec(`INSERT INTO data (x) VALUES (4), (7.0), ('13'), (NULL), (16)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`
		SELECT
			approx_percentile(x, 0.5),
			approx_percentile(x, '[0.25, 0.5, 0.75]'),
			approx_percent_rank(x, 10),
			approx_percent_rank(x, '[0, 4, 6.25, 10, 13.75, 16, 20]'),
			approx_percentile_estimate(approx_percentile_accumulate(x), '[0.25, 0.5, 0.75]'),
			approx_percent_rank_estimate(approx_percentile_accumulate(x), '[0, 4, 6.25, 10, 13.75, 16, 20]')
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	if got := stmt.ColumnFloat(0); got != 10 {
		t.Errorf("got %v, want 10", got)
	}
	var gotPcts []float64
	if err := stmt.ColumnJSON(1, &gotPcts); err != nil {
		t.Error(err)
	} else if !slices.Equal(gotPcts, []float64{6.25, 10, 13.75}) {
		t.Errorf("got %v, want [6.25 10 13.75]", gotPcts)
	}
	if got := stmt.ColumnFloat(2); got != 0.5 {
		t.Errorf("got %v, want 0.5", got)
	}
	var gotRanks []float64
	if err := stmt.ColumnJSON(3, &gotRanks); err != nil {
		t.Error(err)
	} else if !slices.Equal(gotRanks, []float64{0, 0, 0.25, 0.5, 0.75, 1, 1}) {
		t.Errorf("got %v, want [0 0 0.25 0.5 0.75 1 1]", gotRanks)
	}
	if err := stmt.ColumnJSON(4, &gotPcts); err != nil {
		t.Error(err)
	} else if !slices.Equal(gotPcts, []float64{6.25, 10, 13.75}) {
		t.Errorf("got %v, want [6.25 10 13.75]", gotPcts)
	}
	if err := stmt.ColumnJSON(5, &gotRanks); err != nil {
		t.Error(err)
	} else if !slices.Equal(gotRanks, []float64{0, 0, 0.25, 0.5, 0.75, 1, 1}) {
		t.Errorf("got %v, want [0 0 0.25 0.5 0.75 1 1]", gotRanks)
	}
	stmt.Close()

	// Sliding window.
	stmt, _, err = db.Prepare(`
		SELECT
			approx_percentile(x, 0.5) OVER w,
			approx_percentile_estimate(approx_percentile_accumulate(x) OVER w, 0.5),
			json_extract(approx_percentile(x, '[0.5]') OVER w, '$[0]')
		FROM data
		WINDOW w AS (ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING)`)
	if err != nil {
		t.Fatal(err)
	}
	want := [...]float64{5.5, 7, 10, 14.5, 16}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnFloat(0); got != want[i] {
			t.Errorf("row %d: got %v, want %v", i, got, want[i])
		}
		if got := stmt.ColumnFloat(1); got != want[i] {
			t.Errorf("row %d: got %v, want %v", i, got, want[i])
		}
		if got := stmt.ColumnFloat(2); got != want[i] {
			t.Errorf("row %d: got %v, want %v", i, got, want[i])
		}
	}
	stmt.Close()

	// Single row (unit state).
	stmt, _, err = db.Prepare(`
		SELECT
			approx_percentile(x, 0.5),
			approx_percentile(x, '[0.25, 0.5, 0.75]'),
			approx_percent_rank(x, 4),
			approx_percent_rank(x, 5),
			approx_percent_rank(x, '[3, 4, 5]')
		FROM data
		WHERE x < 5`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	if got := stmt.ColumnFloat(0); got != 4 {
		t.Errorf("got %v, want 4", got)
	}
	if err := stmt.ColumnJSON(1, &gotPcts); err != nil {
		t.Error(err)
	} else if !slices.Equal(gotPcts, []float64{4, 4, 4}) {
		t.Errorf("got %v, want [4 4 4]", gotPcts)
	}
	if got := stmt.ColumnFloat(2); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := stmt.ColumnFloat(3); got != 1 {
		t.Errorf("got %v, want 1", got)
	}
	if err := stmt.ColumnJSON(4, &gotRanks); err != nil {
		t.Error(err)
	} else if !slices.Equal(gotRanks, []float64{0, 0, 1}) {
		t.Errorf("got %v, want [0 0 1]", gotRanks)
	}
	stmt.Close()
}

func TestRegister_req_large(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Direct 50,000 element accuracy test.
	stmt, _, err := db.Prepare(`
		SELECT
			approx_percentile(value, '[0.25, 0.5, 0.75, 0.9, 0.99, 0.999, 1.0]'),
			approx_percent_rank(value, '[12500.75, 25000.5, 37500.25, 45000.1, 49500.01, 49950.001, 50000]')
		FROM generate_series(1, 50000)`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	var gotPcts, gotRanks []float64
	if err := stmt.ColumnJSON(0, &gotPcts); err != nil {
		t.Fatal(err)
	}
	if err := stmt.ColumnJSON(1, &gotRanks); err != nil {
		t.Fatal(err)
	}
	stmt.Close()

	q := []float64{0.25, 0.5, 0.75, 0.9, 0.99, 0.999, 1.0}
	tol := []float64{0.025, 0.015, 0.01, 0.005, 0.001, 0.0001, 0}
	for i, p := range q {
		wantVal := 1 + p*49999
		if err := math.Abs(gotPcts[i]-wantVal) / 50000; err > tol[i] {
			t.Errorf("p=%v: got %v, want %v (err %v > %v)", p, gotPcts[i], wantVal, err, tol[i])
		}
		if err := math.Abs(gotRanks[i] - p); err > tol[i] {
			t.Errorf("rank(%v): got %v, want %v (err %v > %v)", wantVal, gotRanks[i], p, err, tol[i])
		}
	}

	// Accumulate / Combine / Window test.
	err = db.Exec(`
		CREATE TABLE sketches (grp, sketch);
		INSERT INTO sketches VALUES (0, NULL);
		INSERT INTO sketches SELECT 1, approx_percentile_accumulate(NULL);
		INSERT INTO sketches
		SELECT value % 4 + 2, approx_percentile_accumulate(value)
		FROM generate_series(1, 50000)
		GROUP BY value % 4;
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`
		SELECT
			approx_percentile_estimate(approx_percentile_combine(sketch) OVER (), 0.5),
			approx_percent_rank_estimate(approx_percentile_combine(sketch) OVER (), 25000.5),
			approx_percentile_estimate(approx_percentile_combine(sketch) OVER (ORDER BY grp ROWS 2 PRECEDING), 0.5)
		FROM sketches`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnFloat(0); math.Abs(got-25000.5)/50000 > 0.02 {
			t.Errorf("row %d: got %.0f, want ~25000.5", i, got)
		}
		if got := stmt.ColumnFloat(1); math.Abs(got-0.5) > 0.02 {
			t.Errorf("row %d: got %.3f, want ~0.5", i, got)
		}
		if i < 2 {
			if got := stmt.ColumnType(2); got != sqlite3.NULL {
				t.Errorf("row %d: got %v, want NULL", i, got)
			}
		} else if got := stmt.ColumnFloat(2); math.Abs(got-25000.5)/50000 > 0.02 {
			t.Errorf("row %d: got %.0f, want ~25000.5", i, got)
		}
	}
	stmt.Close()
}

func TestRegister_req_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []string{
		`SELECT approx_percentile(1, -0.1)`,
		`SELECT approx_percentile(1, 1.1)`,
		`SELECT approx_percentile(1, '')`,
		`SELECT approx_percentile(1, 'null')`,
		`SELECT approx_percentile(1, '[-0.1]')`,
		`SELECT approx_percentile(1, '[1.1]')`,
		`SELECT approx_percent_rank(1, '')`,
		`SELECT approx_percent_rank(1, 'null')`,
		`SELECT approx_percentile_estimate(X'0128', -0.1)`,
		`SELECT approx_percentile_estimate(X'0128', '[1.5]')`,
		`SELECT approx_percentile_estimate('not a blob', 0.5)`,
		`SELECT approx_percentile_estimate(X'00', 0.5)`,
		`SELECT approx_percentile_estimate(X'0228', 0.5)`,
		`SELECT approx_percentile_estimate(X'0110', 0.5)`,
		`SELECT approx_percentile_estimate(X'01280000', 0.5)`,
		`SELECT approx_percentile_estimate(X'01280001', 0.5)`,
		`SELECT approx_percentile_estimate(X'012800010000C07F', 0.5)`,
		`SELECT approx_percentile_combine('not a blob')`,
		`SELECT approx_percentile_combine(X'00')`,
	}
	for _, sql := range tests {
		if err := db.Exec(sql); err == nil {
			t.Errorf("expected error for %q", sql)
		}
	}
}

func Benchmark_req(b *testing.B) {
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
		{"agg_5k", `SELECT approx_percentile(x, 0.5) FROM data WHERE rowid <= 5000`},
		{"agg_50k", `SELECT approx_percentile(x, 0.5) FROM data`},
		{"group_2k_x2", `SELECT approx_percentile(x, 0.5) FROM data WHERE grp2 < 2000 GROUP BY grp2`},
		{"group_1k_x10", `SELECT approx_percentile(x, 0.5) FROM data WHERE grp10 < 1000 GROUP BY grp10`},
		{"win_20", `SELECT approx_percentile(x, 0.5) OVER (ROWS 20 PRECEDING) FROM data WHERE rowid <= 2000`},
		{"win_250", `SELECT approx_percentile(x, 0.5) OVER (ROWS 250 PRECEDING) FROM data WHERE rowid <= 2000`},
		{"win_1000", `SELECT approx_percentile(x, 0.5) OVER (ROWS 1000 PRECEDING) FROM data WHERE rowid <= 2000`},
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
