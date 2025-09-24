package driver

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"math"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_Open_error(t *testing.T) {
	t.Parallel()

	_, err := Open("", nil, nil, nil)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.MISUSE) {
		t.Errorf("got %v, want sqlite3.MISUSE", err)
	}
}

func Test_Open_dir(t *testing.T) {
	t.Parallel()

	db, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Conn(context.TODO())
	if err == nil {
		t.Fatal("want error")
	}
	if !errors.Is(err, sqlite3.CANTOPEN) {
		t.Errorf("got %v, want sqlite3.CANTOPEN", err)
	}
}

func Test_Open_pragma(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t, url.Values{
		"_pragma": {"busy_timeout(1000)"},
	})

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var timeout int
	err = db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout)
	if err != nil {
		t.Fatal(err)
	}
	if timeout != 1000 {
		t.Errorf("got %v, want 1000", timeout)
	}
}

func Test_Open_pragma_invalid(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t, url.Values{
		"_pragma": {"busy_timeout 1000"},
	})

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Conn(context.TODO())
	if err == nil {
		t.Fatal("want error")
	}
	var serr *sqlite3.Error
	if !errors.As(err, &serr) {
		t.Fatalf("got %T, want sqlite3.Error", err)
	}
	if rc := serr.Code(); rc != sqlite3.ERROR {
		t.Errorf("got %d, want sqlite3.ERROR", rc)
	}
	if got := err.Error(); got != `sqlite3: invalid _pragma: sqlite3: SQL logic error: near "1000": syntax error` {
		t.Error("got message:", got)
	}
}

func Test_Open_txLock(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t, url.Values{
		"_txlock": {"exclusive"},
		"_pragma": {"busy_timeout(1000)"},
	})

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tx1, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Begin()
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.BUSY) {
		t.Errorf("got %v, want sqlite3.BUSY", err)
	}
	var terr interface{ Temporary() bool }
	if !errors.As(err, &terr) || !terr.Temporary() {
		t.Error("not temporary", err)
	}

	err = tx1.Commit()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Open_txLock_invalid(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t, url.Values{
		"_txlock": {"xclusive"},
	})

	_, err := Open(tmp)
	if err == nil {
		t.Fatal("want error")
	}
	if got := err.Error(); got != `sqlite3: invalid _txlock: xclusive` {
		t.Error("got message:", got)
	}
}

func Test_BeginTx(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t, url.Values{
		"_txlock": {"exclusive"},
		"_pragma": {"busy_timeout(0)"},
	})

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.BeginTx(t.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err.Error() != string(util.IsolationErr) {
		t.Error("want isolationErr")
	}

	tx1, err := db.BeginTx(t.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	tx2, err := db.BeginTx(t.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx1.Exec(`CREATE TABLE test (col)`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.READONLY) {
		t.Errorf("got %v, want sqlite3.READONLY", err)
	}

	err = tx2.Commit()
	if err != nil {
		t.Fatal(err)
	}

	err = tx1.Commit()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_nested_context(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	outer, err := tx.Query(`SELECT value FROM generate_series(0)`)
	if err != nil {
		t.Fatal(err)
	}
	defer outer.Close()

	want := func(rows *sql.Rows, want int) {
		t.Helper()

		var got int
		rows.Next()
		if err := rows.Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("got %d, want %d", got, want)
		}
	}

	want(outer, 0)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	inner, err := tx.QueryContext(ctx, `SELECT value FROM generate_series(0)`)
	if err != nil {
		t.Fatal(err)
	}
	defer inner.Close()

	want(inner, 0)
	cancel()

	var terr interface{ Temporary() bool }
	if inner.Next() || !errors.Is(inner.Err(), context.Canceled) &&
		(!errors.As(inner.Err(), &terr) || !terr.Temporary()) {
		t.Fatalf("got %v, want cancellation", inner.Err())
	}

	want(outer, 1)
}

func Test_Prepare(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var serr *sqlite3.Error
	_, err = db.Prepare(`SELECT`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.As(err, &serr) {
		t.Fatalf("got %T, want sqlite3.Error", err)
	}
	if rc := serr.Code(); rc != sqlite3.ERROR {
		t.Errorf("got %d, want sqlite3.ERROR", rc)
	}
	if got := err.Error(); got != `sqlite3: SQL logic error: incomplete input` {
		t.Error("got message:", got)
	}

	_, err = db.Prepare(`SELECT 1; `)
	if err != nil {
		t.Error(err)
	}

	_, err = db.Prepare(`SELECT 1; SELECT`)
	if err.Error() != string(util.TailErr) {
		t.Error("want tailErr")
	}

	_, err = db.Prepare(`SELECT 1; SELECT 2`)
	if err.Error() != string(util.TailErr) {
		t.Error("want tailErr")
	}
}

func Test_QueryRow_named(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(t.Context(), `SELECT ?, ?5, :AAA, @AAA, $AAA`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	date := time.Now()
	row := stmt.QueryRow(true, sql.Named("AAA", math.Pi), nil /*3*/, nil /*4*/, date /*5*/)

	var first bool
	var fifth time.Time
	var colon, at, dollar float32
	err = row.Scan(&first, &fifth, &colon, &at, &dollar)
	if err != nil {
		t.Fatal(err)
	}

	if first != true {
		t.Errorf("want true, got %v", first)
	}
	if colon != math.Pi {
		t.Errorf("want π, got %v", colon)
	}
	if at != math.Pi {
		t.Errorf("want π, got %v", at)
	}
	if dollar != math.Pi {
		t.Errorf("want π, got %v", dollar)
	}
	if !fifth.Equal(date) {
		t.Errorf("want %v, got %v", date, fifth)
	}
}

func Test_QueryRow_blob_null(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT NULL    UNION ALL
		SELECT x'cafe' UNION ALL
		SELECT x'babe' UNION ALL
		SELECT NULL
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	want := [][]byte{nil, {0xca, 0xfe}, {0xba, 0xbe}, nil}
	for i := 0; rows.Next(); i++ {
		var buf sql.RawBytes
		err = rows.Scan(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf, want[i]) {
			t.Errorf("got %q, want %q", buf, want[i])
		}
	}
}

func Test_time(t *testing.T) {
	t.Parallel()

	for _, fmt := range []string{"auto", "sqlite", "rfc3339", time.ANSIC} {
		t.Run(fmt, func(t *testing.T) {
			tmp := memdb.TestDB(t, url.Values{
				"_timefmt": {fmt},
			})

			db, err := Open(tmp)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			twosday := time.Date(2022, 2, 22, 22, 22, 22, 0, time.UTC)

			_, err = db.Exec(`CREATE TABLE test (at DATETIME)`)
			if err != nil {
				t.Fatal(err)
			}

			_, err = db.Exec(`INSERT INTO test VALUES (?)`, twosday)
			if err != nil {
				t.Fatal(err)
			}

			var got time.Time
			err = db.QueryRow(`SELECT * FROM test`).Scan(&got)
			if err != nil {
				t.Fatal(err)
			}

			if !got.Equal(twosday) {
				t.Errorf("got: %v", got)
			}
		})
	}
}

func Test_ColumnType_ScanType(t *testing.T) {
	var (
		INT  = reflect.TypeFor[int64]()
		REAL = reflect.TypeFor[float64]()
		TEXT = reflect.TypeFor[string]()
		BLOB = reflect.TypeFor[[]byte]()
		BOOL = reflect.TypeFor[bool]()
		TIME = reflect.TypeFor[time.Time]()
		ANY  = reflect.TypeFor[any]()
	)

	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE test (
			col_int  INTEGER,
			col_real REAL,
			col_text TEXT,
			col_blob BLOB,
			col_bool BOOLEAN,
			col_time DATETIME,
			col_decimal DECIMAL
		);
		INSERT INTO test VALUES
			(1, 1, 1, 1, 1, 1, 1),
			(2.0, 2.0, 2.0, 2.0, 2.0, 2.0, 2.0),
			('1', '1', '1', '1', '1', '1', '1'),
			('x', 'x', 'x', 'x', 'x', 'x', 'x'),
			(x'', x'', x'', x'', x'', x'', x''),
			('2006-01-02T15:04:05Z', '2006-01-02T15:04:05Z', '2006-01-02T15:04:05Z', '2006-01-02T15:04:05Z',
			 '2006-01-02T15:04:05Z', '2006-01-02T15:04:05Z', '2006-01-02T15:04:05Z'),
			(TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE),
			(NULL, NULL, NULL, NULL, NULL, NULL, NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`SELECT * FROM test`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}

	want := [][]reflect.Type{
		{INT, REAL, TEXT, BLOB, BOOL, TIME, ANY},
		{INT, REAL, TEXT, INT, BOOL, TIME, INT},
		{INT, REAL, TEXT, REAL, INT, TIME, INT},
		{INT, REAL, TEXT, TEXT, BOOL, TIME, INT},
		{TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT},
		{BLOB, BLOB, BLOB, BLOB, BLOB, BLOB, BLOB},
		{TEXT, TEXT, TEXT, TEXT, TEXT, TIME, TEXT},
		{INT, REAL, TEXT, INT, BOOL, TIME, INT},
		{ANY, ANY, ANY, BLOB, ANY, ANY, ANY},
	}
	for j, c := range cols {
		got := c.ScanType()
		if got != want[0][j] {
			t.Errorf("want %v, got %v, at column %d", want[0][j], got, j)
		}
	}

	dest := make([]any, len(cols))
	for i := 1; rows.Next(); i++ {
		cols, err := rows.ColumnTypes()
		if err != nil {
			t.Fatal(err)
		}

		for j, c := range cols {
			got := c.ScanType()
			if got != want[i][j] {
				t.Errorf("want %v, got %v, at row %d column %d", want[i][j], got, i, j)
			}
			dest[j] = reflect.New(got).Interface()
		}

		err = rows.Scan(dest...)
		if err != nil {
			t.Error(err)
		}
	}

	err = rows.Err()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_rows_ScanColumn(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var tm time.Time
	err = db.QueryRow(`SELECT NULL`).Scan(&tm)
	if err == nil {
		t.Error("want error")
	}
	// Go 1.26
	err = db.QueryRow(`SELECT datetime()`).Scan(&tm)
	if err != nil && !strings.HasPrefix(err.Error(), "sql: Scan error") {
		t.Error(err)
	}

	var nt sql.NullTime
	err = db.QueryRow(`SELECT NULL`).Scan(&nt)
	if err != nil {
		t.Error(err)
	}
	// Go 1.26
	err = db.QueryRow(`SELECT datetime()`).Scan(&nt)
	if err != nil && !strings.HasPrefix(err.Error(), "sql: Scan error") {
		t.Error(err)
	}
}

func Benchmark_loop(b *testing.B) {
	db, err := Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var version string
	err = db.QueryRow(`SELECT sqlite_version();`).Scan(&version)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, err := db.ExecContext(b.Context(),
			`WITH RECURSIVE c(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM c WHERE x < 1000000) SELECT x FROM c;`)
		if err != nil {
			b.Fatal(err)
		}
	}
}
