package approx_test

import (
	"math"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_info(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// NULL input.
	stmt, _, err := db.Prepare(`SELECT approx_sketch_info(NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	if got := stmt.ColumnType(0); got != sqlite3.NULL {
		t.Errorf("got %v, want NULL", got)
	}
	stmt.Close()

	// Exact / empty / edge-case sketches and JSON subtype.
	exactTests := []struct {
		sql, want string
	}{
		// Empty sketches (0 rows).
		{
			`SELECT approx_sketch_info(approx_count_distinct_accumulate(value)) FROM generate_series(1, 0)`,
			`{"type":"count_distinct","count":0,"stderr":0.0,"rse":0.0}`,
		},
		{
			`SELECT approx_sketch_info(approx_percentile_accumulate(value)) FROM generate_series(1, 0)`,
			`{"type":"percentile","count":0,"min":null,"max":null,"avg":null,"stddev":null,"rank_rse":0.0}`,
		},
		{
			`SELECT approx_sketch_info(approx_top_k_accumulate(value, 10)) FROM generate_series(1, 0)`,
			`{"type":"top_k","k":null,"total":0,"max_error":0}`,
		},
		// All-NULL rows.
		{
			`SELECT approx_sketch_info(approx_top_k_accumulate(NULL, 10))`,
			`{"type":"top_k","k":10,"total":0,"max_error":0}`,
		},
		// Exact HLL (EXPLICIT storage <= 160 items).
		{
			`SELECT approx_sketch_info(approx_count_distinct_accumulate(value)) FROM generate_series(1, 50)`,
			`{"type":"count_distinct","count":50,"stderr":0.0,"rse":0.0}`,
		},
		// Single-row REQ (32-bit formatting + null sample stddev) and multi-row uncompacted REQ.
		{
			`SELECT approx_sketch_info(approx_percentile_accumulate(0.1))`,
			`{"type":"percentile","count":1,"min":0.1,"max":0.1,"avg":0.1,"stddev":null,"rank_rse":0.0}`,
		},
		{
			`SELECT approx_sketch_info(approx_percentile_accumulate(value)) FROM generate_series(1, 4)`,
			`{"type":"percentile","count":4,"min":1.0,"max":4.0,"avg":2.5,"stddev":1.2909944487358056,"rank_rse":0.0}`,
		},
		// REQ with ±Inf.
		{
			`SELECT approx_sketch_info(approx_percentile_accumulate(column1)) FROM (VALUES (1.0), (1e999))`,
			`{"type":"percentile","count":2,"min":1.0,"max":9.0e999,"avg":9.0e999,"stddev":null,"rank_rse":0.0}`,
		},
		{
			`SELECT approx_sketch_info(approx_percentile_accumulate(column1)) FROM (VALUES (-1e999), (1e999))`,
			`{"type":"percentile","count":2,"min":-9.0e999,"max":9.0e999,"avg":null,"stddev":null,"rank_rse":0.0}`,
		},
		// Uncompacted FI (exact, max_error = 0) and JSON subtype embedding.
		{
			`SELECT json_object('info', approx_sketch_info(approx_top_k_accumulate(value, 1.5, 5))) FROM generate_series(1, 10)`,
			`{"info":{"type":"top_k","k":5,"total":15,"max_error":0}}`,
		},
	}
	for _, tc := range exactTests {
		stmt, _, err := db.Prepare(tc.sql)
		if err != nil {
			t.Fatal(err)
		}
		if !stmt.Step() {
			t.Fatal(stmt.Err())
		}
		if got := stmt.ColumnText(0); got != tc.want {
			t.Errorf("%q:\ngot  %s\nwant %s", tc.sql, got, tc.want)
		}
		stmt.Close()
	}

	// Large sketches: SPARSE & FULL HLL, compacted REQ, compacted FI.
	stmt, _, err = db.Prepare(`
		SELECT
			(SELECT approx_sketch_info(approx_count_distinct_accumulate(value)) FROM generate_series(1, 500)),
			(SELECT approx_sketch_info(approx_count_distinct_accumulate(value)) FROM generate_series(1, 25000)),
			(SELECT approx_sketch_info(approx_percentile_accumulate(value)) FROM generate_series(1, 50000)),
			(SELECT approx_sketch_info(approx_top_k_accumulate(value % 500, 1.5, 5)) FROM generate_series(1, 50000))
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}

	var hllSparse, hllFull struct {
		Type   string  `json:"type"`
		Count  int64   `json:"count"`
		Stderr float64 `json:"stderr"`
		RSE    float64 `json:"rse"`
	}
	if err := stmt.ColumnJSON(0, &hllSparse); err != nil {
		t.Fatal(err)
	}
	if hllSparse.Type != "count_distinct" || math.Abs(float64(hllSparse.Count)-500) > 25 || !(0.01 < hllSparse.RSE && hllSparse.RSE < 0.023) {
		t.Errorf("hllSparse: got %+v", hllSparse)
	}
	if err := stmt.ColumnJSON(1, &hllFull); err != nil {
		t.Fatal(err)
	}
	if hllFull.Type != "count_distinct" || math.Abs(float64(hllFull.Count)-25000) > 1500 || math.Abs(hllFull.RSE-0.02296) > 1e-4 {
		t.Errorf("hllFull: got %+v", hllFull)
	}

	var reqLarge struct {
		Type    string  `json:"type"`
		Count   int64   `json:"count"`
		Min     float64 `json:"min"`
		Max     float64 `json:"max"`
		Avg     float64 `json:"avg"`
		Stddev  float64 `json:"stddev"`
		RankRSE float64 `json:"rank_rse"`
	}
	if err := stmt.ColumnJSON(2, &reqLarge); err != nil {
		t.Fatal(err)
	}
	wantStd := math.Sqrt(50000.0 * 50001.0 / 12.0)
	if reqLarge.Type != "percentile" || reqLarge.Count != 50000 || reqLarge.Min != 1 || reqLarge.Max != 50000 ||
		math.Abs(reqLarge.Avg-25000.5)/25000.5 > 0.01 ||
		math.Abs(reqLarge.Stddev-wantStd)/wantStd > 0.02 ||
		!(0.003 < reqLarge.RankRSE && reqLarge.RankRSE < 0.004) {
		t.Errorf("reqLarge: got %+v", reqLarge)
	}

	var topLarge struct {
		Type     string  `json:"type"`
		K        int     `json:"k"`
		Total    float64 `json:"total"`
		MaxError float64 `json:"max_error"`
	}
	if err := stmt.ColumnJSON(3, &topLarge); err != nil {
		t.Fatal(err)
	}
	if topLarge.Type != "top_k" || topLarge.K != 5 || topLarge.Total != 75000 ||
		!(0 < topLarge.MaxError && topLarge.MaxError <= 75000.0/129.0) {
		t.Errorf("topLarge: got %+v", topLarge)
	}
	stmt.Close()

	// Error cases.
	errTests := []string{
		`SELECT approx_sketch_info('not a blob')`,
		`SELECT approx_sketch_info(123)`,
		`SELECT approx_sketch_info(X'')`,
		`SELECT approx_sketch_info(X'00')`,
		`SELECT approx_sketch_info(X'17')`,
		`SELECT approx_sketch_info(X'11')`,
		`SELECT approx_sketch_info(X'15')`,
		`SELECT approx_sketch_info(X'1601')`,
	}
	for _, sql := range errTests {
		if err := db.Exec(sql); err == nil {
			t.Errorf("expected error for %q", sql)
		}
	}
}
