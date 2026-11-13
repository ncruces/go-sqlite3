package approx_test

import (
	"math"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func TestRegister_top(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE data (x, w)`)
	if err != nil {
		t.Fatal(err)
	}

	// Empty table.
	stmt, _, err := db.Prepare(`
		SELECT
			approx_top_k(x, 5),
			approx_top_k(x, w, 5),
			approx_top_k_estimate(approx_top_k_accumulate(x, 5)),
			approx_top_k_estimate(approx_top_k_accumulate(x, w, 5), 50),
			approx_top_k_combine(x),
			approx_top_k_estimate(NULL),
			approx_top_k_estimate(NULL, 5)
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

	// Mixed types, normalization (0 == -0.0, 1 == 1.0, large ints), weights, ignored NULL/0/non-numeric weights, tie-breaking.
	err = db.Exec(`
		INSERT INTO data (x, w) VALUES
			(NULL, 10),
			('skip_null_w', NULL),
			('skip_zero_w', 0),
			('skip_text_w', 'not_num'),
			(0, 1.5),
			(-0.0, '2.5'),
			(1, 2),
			(1.0, 2),
			('1', 4),
			('', 4),
			(1.5, 4),
			(9007199254740992, 1),
			(9007199254740993, 3),
			(9007199254740992.0, 2)
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`
		SELECT
			approx_top_k(x, 4),
			approx_top_k(x, w, 6),
			approx_top_k_estimate(approx_top_k_accumulate(x, w, 6)),
			approx_top_k_estimate(approx_top_k_accumulate(x, w, 6), 3),
			json_object('top', approx_top_k(x, w, 2))
		FROM data`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	// Counts (unweighted): 0 -> 2, 1 -> 2, 9007199254740992 -> 2; ties at 1 broken by 1.5 < 9007199254740993 < "" < "1" < ...
	const wantCount4 = `[[0,2],[1,2],[9007199254740992,2],[1.5,1]]`
	if got := stmt.ColumnText(0); got != wantCount4 {
		t.Errorf("col 0: got %s, want %s", got, wantCount4)
	}
	// Weighted sums: 0 -> 4, 1 -> 4, 1.5 -> 4, "" -> 4, "1" -> 4 (sorted: 0 < 1 < 1.5 < "" < "1"), then 9007199254740992 -> 3, 9007199254740993 -> 3.
	const wantSum6 = `[[0,4],[1,4],[1.5,4],["",4],["1",4],[9007199254740992,3]]`
	if got := stmt.ColumnText(1); got != wantSum6 {
		t.Errorf("col 1: got %s, want %s", got, wantSum6)
	}
	if got := stmt.ColumnText(2); got != wantSum6 {
		t.Errorf("col 2: got %s, want %s", got, wantSum6)
	}
	const wantSum3 = `[[0,4],[1,4],[1.5,4]]`
	if got := stmt.ColumnText(3); got != wantSum3 {
		t.Errorf("col 3: got %s, want %s", got, wantSum3)
	}
	const wantNested = `{"top":[[0,4],[1,4]]}`
	if got := stmt.ColumnText(4); got != wantNested {
		t.Errorf("col 4: got %s, want %s", got, wantNested)
	}
	stmt.Close()

	// Single-key unit states (int, float, empty text, repeated text) + JSON subtype.
	qASCII, _ := util.AppendQuote(nil, "a<b")
	qUTF8, _ := util.AppendQuote(nil, "a<bπ")
	if strings.Contains(string(qASCII), "<") != strings.Contains(string(qUTF8), "<") {
		t.Errorf("inconsistent < escaping: %s vs %s", qASCII, qUTF8)
	}
	unitTests := []struct {
		sql, want string
	}{
		{`SELECT json_array(approx_top_k(42, 5))`, `[[[42,1]]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate(42, 5))`, `[[42,1]]`},
		{`SELECT approx_top_k(2.5, 1.25, 5)`, `[[2.5,1.25]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate(2.5, 1.25, 5))`, `[[2.5,1.25]]`},
		{`SELECT approx_top_k('', 5)`, `[["",1]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate('', 5))`, `[["",1]]`},
		{`SELECT approx_top_k('a<b', 5)`, `[[` + string(qASCII) + `,1]]`},
		{`SELECT approx_top_k('a<bπ', 5)`, `[[` + string(qUTF8) + `,1]]`},
		{`SELECT approx_top_k('a"b\c' || char(1, 8, 9, 10, 12, 13) || 'π', 5)`, `[["a\"b\\c\u0001\b\t\n\f\rπ",1]]`},
		{`SELECT approx_top_k(1e999, 5)`, `[[9.0e999,1]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate(1e999, 5))`, `[[9.0e999,1]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate(-1e999, 5))`, `[[-9.0e999,1]]`},
		{`SELECT approx_top_k(1, 1e308, 5) FROM generate_series(1, 2)`, `[[1,9.0e999]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate(1, 1e308, 5)) FROM generate_series(1, 2)`, `[[1,9.0e999]]`},
		{`SELECT approx_top_k_estimate(approx_top_k_accumulate(value % 256, 1e308, 5)) FROM generate_series(0, 511)`, `[]`},
		{`SELECT approx_top_k('a', 5) FROM generate_series(1, 3)`, `[["a",3]]`},
		{`SELECT approx_top_k(value, 5) FROM (SELECT 1.5 AS value UNION ALL SELECT 1.5)`, `[[1.5,2]]`},
	}
	for _, tc := range unitTests {
		stmt, _, err := db.Prepare(tc.sql)
		if err != nil {
			t.Fatal(err)
		}
		if !stmt.Step() {
			t.Fatal(stmt.Err())
		}
		if got := stmt.ColumnText(0); got != tc.want {
			t.Errorf("%q: got %s, want %s", tc.sql, got, tc.want)
		}
		stmt.Close()
	}

	// Invalid UTF-8 TEXT follows Go json v1 (replacement) vs v2 (error).
	_, wantUTF8Err := util.AppendQuote(nil, "a\xff")
	for _, sql := range []string{
		`SELECT approx_top_k(CAST(X'61ff' AS TEXT), 5)`,
		`SELECT approx_top_k(x, 5) FROM (SELECT 'a' AS x UNION ALL SELECT CAST(X'61ff' AS TEXT))`,
	} {
		err := db.Exec(sql)
		if (err != nil) != (wantUTF8Err != nil) {
			t.Errorf("%q: got err=%v, want err=%v", sql, err, wantUTF8Err)
		}
	}

	// Sliding window over small sequence.
	err = db.Exec(`
		CREATE TABLE seq (x);
		INSERT INTO seq VALUES ('a'), ('b'), ('a'), (NULL), ('b'), ('b'), ('c');
	`)
	if err != nil {
		t.Fatal(err)
	}
	stmt, _, err = db.Prepare(`
		SELECT
			approx_top_k(x, 2) OVER w,
			approx_top_k_estimate(approx_top_k_accumulate(x, 2) OVER w)
		FROM seq
		WINDOW w AS (ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING)`)
	if err != nil {
		t.Fatal(err)
	}
	wantWin := [...]string{
		`[["a",1],["b",1]]`,
		`[["a",2],["b",1]]`,
		`[["a",1],["b",1]]`,
		`[["a",1],["b",1]]`,
		`[["b",2]]`,
		`[["b",2],["c",1]]`,
		`[["b",1],["c",1]]`,
	}
	for i := 0; stmt.Step(); i++ {
		if got := stmt.ColumnText(0); got != wantWin[i] {
			t.Errorf("row %d col 0: got %s, want %s", i, got, wantWin[i])
		}
		if got := stmt.ColumnText(1); got != wantWin[i] {
			t.Errorf("row %d col 1: got %s, want %s", i, got, wantWin[i])
		}
	}
	stmt.Close()
}

func TestRegister_top_large(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 50,000 rows: 25 heavy hitters ('k1'..'k25' with counts 2500, 2400, ..., 100 = 32,500 rows)
	// plus 17,500 distinct singleton tail items.
	err = db.Exec(`
		CREATE TABLE stream (id INT, item TEXT, w REAL);
		WITH RECURSIVE hh(i, lo, hi) AS (
			SELECT 1, 1, 2500
			UNION ALL
			SELECT i + 1, hi + 1, hi + (26 - (i + 1)) * 100
			FROM hh WHERE i < 25
		)
		INSERT INTO stream
		SELECT s.value,
			COALESCE((SELECT 'k' || hh.i FROM hh WHERE s.value BETWEEN hh.lo AND hh.hi), 'tail_' || s.value),
			1.5
		FROM generate_series(1, 50000) AS s;
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT
			approx_top_k(item, 5),
			approx_top_k(item, w, 5)
		FROM stream`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	var gotCounts, gotSums [][2]any
	if err := stmt.ColumnJSON(0, &gotCounts); err != nil {
		t.Fatal(err)
	}
	if err := stmt.ColumnJSON(1, &gotSums); err != nil {
		t.Fatal(err)
	}
	stmt.Close()

	wantKeys := []string{"k1", "k2", "k3", "k4", "k5"}
	wantCounts := []float64{2500, 2400, 2300, 2200, 2100}
	// With K=5, C=128, max theoretical error <= 50000 / 129 ≈ 388.
	const maxCountErr = 50000.0 / 129.0
	if len(gotCounts) != 5 || len(gotSums) != 5 {
		t.Fatalf("got lengths %d/%d, want 5/5", len(gotCounts), len(gotSums))
	}
	for i := range 5 {
		if gotCounts[i][0] != wantKeys[i] {
			t.Errorf("count[%d] key: got %v, want %v", i, gotCounts[i][0], wantKeys[i])
		}
		estC := gotCounts[i][1].(float64)
		if estC < wantCounts[i] || estC-wantCounts[i] > maxCountErr {
			t.Errorf("count[%d]: got %v, want in [%v, %v]", i, estC, wantCounts[i], wantCounts[i]+maxCountErr)
		}

		if gotSums[i][0] != wantKeys[i] {
			t.Errorf("sum[%d] key: got %v, want %v", i, gotSums[i][0], wantKeys[i])
		}
		wantW := wantCounts[i] * 1.5
		estW := gotSums[i][1].(float64)
		if estW < wantW-1e-9 || estW-wantW > maxCountErr*1.5+1e-9 {
			t.Errorf("sum[%d]: got %v, want in [%v, %v]", i, estW, wantW, wantW+maxCountErr*1.5)
		}
	}

	// Accumulate / Combine across shards with DIFFERENT K (and C), empty sketch, and NULL sketch.
	err = db.Exec(`
		CREATE TABLE sketches (grp INT, sketch BLOB);
		INSERT INTO sketches VALUES (0, NULL);
		INSERT INTO sketches SELECT 1, approx_top_k_accumulate(item, 10) FROM stream WHERE id < 0;
		INSERT INTO sketches SELECT 2, approx_top_k_accumulate(NULL, 10);
		INSERT INTO sketches SELECT 3, approx_top_k_accumulate(item, 5) FROM stream WHERE id % 2 = 0;
		INSERT INTO sketches SELECT 4, approx_top_k_accumulate(item, 50) FROM stream WHERE id % 2 = 1;
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Combined sketch has min K = 5, min C = 128; 2-arg estimate allows querying up to C = 128 (e.g. K = 20).
	stmt, _, err = db.Prepare(`
		SELECT
			approx_top_k_estimate(approx_top_k_combine(sketch) OVER ()),
			approx_top_k_estimate(approx_top_k_combine(sketch) OVER (), 20),
			approx_top_k_estimate(approx_top_k_combine(sketch) OVER (ORDER BY grp ROWS 2 PRECEDING))
		FROM sketches
		ORDER BY grp`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; stmt.Step(); i++ {
		var rowTop5, rowTop20 [][2]any
		if err := stmt.ColumnJSON(0, &rowTop5); err != nil {
			t.Fatal(err)
		}
		if len(rowTop5) != 5 || rowTop5[0][0] != "k1" {
			t.Errorf("row %d top5: got %v", i, rowTop5)
		}
		if err := stmt.ColumnJSON(1, &rowTop20); err != nil {
			t.Fatal(err)
		}
		if len(rowTop20) != 20 || rowTop20[0][0] != "k1" {
			t.Errorf("row %d top20: got len %d", i, len(rowTop20))
		}
		if i < 3 {
			if got := stmt.ColumnType(2); got != sqlite3.NULL {
				t.Errorf("row %d window: got %v, want NULL", i, got)
			}
		} else {
			var winTop [][2]any
			if err := stmt.ColumnJSON(2, &winTop); err != nil {
				t.Fatal(err)
			}
			if len(winTop) != 5 || winTop[0][0] != "k1" {
				t.Errorf("row %d window: got %v", i, winTop)
			}
		}
	}
	stmt.Close()

	// All-singleton boundary purge (256 distinct items, C = 128 -> count = 0, offset = 1 -> "[]").
	stmt, _, err = db.Prepare(`
		SELECT
			approx_top_k(value, 5),
			approx_top_k_estimate(approx_top_k_accumulate(value, 5))
		FROM generate_series(1, 256)`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	if got := stmt.ColumnText(0); got != `[]` {
		t.Errorf("got %q, want []", got)
	}
	if got := stmt.ColumnText(1); got != `[]` {
		t.Errorf("got %q, want []", got)
	}
	stmt.Close()

	// MaxInt64 vs 2^63 float ordering check (both orders).
	stmt, _, err = db.Prepare(`
		SELECT
			approx_top_k(x, 4),
			approx_top_k_estimate(approx_top_k_combine(s))
		FROM (
			SELECT 9223372036854775807 AS x, approx_top_k_accumulate(value, 5) AS s FROM generate_series(1, 300)
			UNION ALL
			SELECT 9223372036854775808.0, approx_top_k_accumulate(value + 0.5, 1.25, 5) FROM generate_series(1, 300)
			UNION ALL
			SELECT 1.5, approx_top_k_accumulate(1, 5)
			UNION ALL
			SELECT 2, approx_top_k_accumulate(2.5, 1.25, 5)
		)`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	}
	var gotBig [][2]float64
	if err := stmt.ColumnJSON(0, &gotBig); err != nil {
		t.Fatal(err)
	}
	if len(gotBig) != 4 || gotBig[0][0] != 1.5 || gotBig[1][0] != 2 || math.IsNaN(gotBig[2][0]) {
		t.Errorf("got %v", gotBig)
	}
	if got := stmt.ColumnType(1); got != sqlite3.TEXT {
		t.Errorf("got %v, want TEXT", got)
	}
	stmt.Close()

	// Sliding window over >2 distinct items (exercises unit + summary Combine in twoStack.Inverse).
	stmt, _, err = db.Prepare(`
		SELECT approx_top_k(CASE WHEN value % 3 = 0 THEN value + 0.5 ELSE value END, 5)
			OVER (ROWS BETWEEN 5 PRECEDING AND 1 FOLLOWING)
		FROM generate_series(1, 30)`)
	if err != nil {
		t.Fatal(err)
	}
	for stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.TEXT {
			t.Errorf("got %v, want TEXT", got)
		}
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}
	stmt.Close()
}

func TestRegister_top_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []string{
		`SELECT approx_top_k(1, 0)`,
		`SELECT approx_top_k(1, -1)`,
		`SELECT approx_top_k(1, 1001)`,
		`SELECT approx_top_k(1, NULL)`,
		`SELECT approx_top_k(1, 'abc')`,
		`SELECT approx_top_k(NULL, 0)`,
		`SELECT approx_top_k(X'01', 5)`,
		`SELECT approx_top_k(1, -0.5, 5)`,
		`SELECT approx_top_k(1, 1e999, 5)`,
		`SELECT approx_top_k_accumulate(1, 0)`,
		`SELECT approx_top_k_accumulate(1, 1001)`,
		`SELECT approx_top_k_accumulate(X'01', 5)`,
		`SELECT approx_top_k_accumulate(1, -1, 5)`,
		`SELECT approx_top_k_estimate('not a blob')`,
		`SELECT approx_top_k_estimate('not a blob', 5)`,
		`SELECT approx_top_k_estimate(X'00')`,
		`SELECT approx_top_k_estimate(X'16')`,
		`SELECT approx_top_k_estimate(X'160001')`,                         // trailing byte after K=0
		`SELECT approx_top_k_estimate(X'1600', 0)`,                        // K=0 on empty sketch
		`SELECT approx_top_k_estimate(X'1600', 4001)`,                     // K>topMaxC on empty sketch
		`SELECT approx_top_k_estimate(X'16EA07800100')`,                   // K=1002 > topMaxK
		`SELECT approx_top_k_estimate(X'160A0500')`,                       // C=5 < K=10
		`SELECT approx_top_k_estimate(X'1601A11F00')`,                     // C=4001 > topMaxC
		`SELECT approx_top_k_estimate(X'160180018202')`,                   // count=129 > C=128
		`SELECT approx_top_k_estimate(X'1601800101')`,                     // hasOffset=1, missing offset
		`SELECT approx_top_k_estimate(X'16018001010000000000000000')`,     // hasOffset=1, offset=0
		`SELECT approx_top_k_estimate(X'1601800102080001')`,               // reserved tag bit set (0x08)
		`SELECT approx_top_k_estimate(X'1601800102030001')`,               // invalid keyType=3
		`SELECT approx_top_k_estimate(X'1601800102000200')`,               // integer weight u=0
		`SELECT approx_top_k_estimate(X'160180010201000000000000F03F01')`, // integral float 1.0 in FLOAT key
		`SELECT approx_top_k_estimate(X'1601800104000201000201')`,         // duplicate int key 1
		`SELECT approx_top_k_estimate(approx_top_k_accumulate(1, 5), 0)`,
		`SELECT approx_top_k_estimate(approx_top_k_accumulate(1, 5), 129)`, // > C=128
		`SELECT approx_top_k_estimate(approx_count_distinct_accumulate(1))`,
		`SELECT approx_top_k_estimate(approx_percentile_accumulate(1))`,
		`SELECT approx_count_distinct_estimate(approx_top_k_accumulate(1, 5))`,
		`SELECT approx_percentile_estimate(approx_top_k_accumulate(1, 5), 0.5)`,
		`SELECT approx_top_k_combine('not a blob')`,
		`SELECT approx_top_k_combine(X'00')`,
		`SELECT approx_top_k_combine(approx_count_distinct_accumulate(1))`,
		`SELECT approx_top_k_combine(approx_percentile_accumulate(1))`,
	}
	for _, sql := range tests {
		if err := db.Exec(sql); err == nil {
			t.Errorf("expected error for %q", sql)
		}
	}
}

func Benchmark_top(b *testing.B) {
	db, err := sqlite3.OpenContext(b.Context(), ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE data (grp10 INT, grp2 INT, x INT);
		INSERT INTO data SELECT value / 10, value / 2, value % 500 FROM generate_series(0, 49999);
		CREATE INDEX idx_grp10 ON data (grp10, x);
		CREATE INDEX idx_grp2 ON data (grp2, x);
	`)
	if err != nil {
		b.Fatal(err)
	}

	queries := []struct {
		name, sql string
	}{
		{"agg_5k", `SELECT approx_top_k(x, 10) FROM data WHERE rowid <= 5000`},
		{"agg_50k", `SELECT approx_top_k(x, 10) FROM data`},
		{"group_2k_x2", `SELECT approx_top_k(x, 10) FROM data WHERE grp2 < 2000 GROUP BY grp2`},
		{"group_1k_x10", `SELECT approx_top_k(x, 10) FROM data WHERE grp10 < 1000 GROUP BY grp10`},
		{"win_20", `SELECT approx_top_k(x, 10) OVER (ROWS 20 PRECEDING) FROM data WHERE rowid <= 2000`},
		{"win_250", `SELECT approx_top_k(x, 10) OVER (ROWS 250 PRECEDING) FROM data WHERE rowid <= 2000`},
		{"win_1000", `SELECT approx_top_k(x, 10) OVER (ROWS 1000 PRECEDING) FROM data WHERE rowid <= 2000`},
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
