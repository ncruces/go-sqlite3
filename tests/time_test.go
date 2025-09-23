package tests

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestTimeFormat_Encode(t *testing.T) {
	t.Parallel()

	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))

	tests := []struct {
		fmt  sqlite3.TimeFormat
		time time.Time
		want any
	}{
		{sqlite3.TimeFormatDefault, reference, "2013-10-07T04:23:19.12-04:00"},
		{sqlite3.TimeFormatJulianDay, reference, 2456572.849526851851852},
		{sqlite3.TimeFormatUnix, reference, int64(1381134199)},
		{sqlite3.TimeFormatUnixFrac, reference, 1381134199.120},
		{sqlite3.TimeFormatUnixMilli, reference, int64(1381134199_120)},
		{sqlite3.TimeFormatUnixMicro, reference, int64(1381134199_120000)},
		{sqlite3.TimeFormatUnixNano, reference, int64(1381134199_120000000)},
		{sqlite3.TimeFormat7, reference, "2013-10-07T08:23:19.120"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := tt.fmt.Encode(tt.time); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%q.Encode(%v) = %v, want %v", tt.fmt, tt.time, got, tt.want)
			}
		})
	}
}

func TestTimeFormat_Decode(t *testing.T) {
	t.Parallel()

	const offset = -4 * 3600
	zone := time.FixedZone("", offset)
	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, zone)
	refnodate := time.Date(2000, 01, 1, 4, 23, 19, 120_000_000, zone)

	tests := []struct {
		fmt       sqlite3.TimeFormat
		val       any
		want      time.Time
		wantDelta time.Duration
		wantOff   int
		wantErr   bool
	}{
		{sqlite3.TimeFormatJulianDay, "2456572.849526851851852", reference, 0, 0, false},
		{sqlite3.TimeFormatJulianDay, 2456572.849526851851852, reference, time.Millisecond, 0, false},
		{sqlite3.TimeFormatJulianDay, int64(2456572), reference, 24 * time.Hour, 0, false},
		{sqlite3.TimeFormatJulianDay, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormatUnix, "1381134199.120", reference, time.Microsecond, 0, false},
		{sqlite3.TimeFormatUnix, 1381134199.120, reference, time.Microsecond, 0, false},
		{sqlite3.TimeFormatUnix, int64(1381134199), reference, time.Second, 0, false},
		{sqlite3.TimeFormatUnix, "abc", time.Time{}, 0, 0, true},
		{sqlite3.TimeFormatUnix, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormatUnixMilli, "1381134199120", reference, 0, 0, false},
		{sqlite3.TimeFormatUnixMilli, 1381134199.120e3, reference, 0, 0, false},
		{sqlite3.TimeFormatUnixMilli, int64(1381134199_120), reference, 0, 0, false},
		{sqlite3.TimeFormatUnixMilli, "abc", time.Time{}, 0, 0, true},
		{sqlite3.TimeFormatUnixMilli, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormatUnixMicro, "1381134199120000", reference, 0, 0, false},
		{sqlite3.TimeFormatUnixMicro, 1381134199.120e6, reference, 0, 0, false},
		{sqlite3.TimeFormatUnixMicro, int64(1381134199_120000), reference, 0, 0, false},
		{sqlite3.TimeFormatUnixMicro, "abc", time.Time{}, 0, 0, true},
		{sqlite3.TimeFormatUnixMicro, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormatUnixNano, "1381134199120000000", reference, 0, 0, false},
		{sqlite3.TimeFormatUnixNano, 1381134199.120e9, reference, 0, 0, false},
		{sqlite3.TimeFormatUnixNano, int64(1381134199_120000000), reference, 0, 0, false},
		{sqlite3.TimeFormatUnixNano, "abc", time.Time{}, 0, 0, true},
		{sqlite3.TimeFormatUnixNano, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormatAuto, "2456572.849526851851852", reference, time.Millisecond, 0, false},
		{sqlite3.TimeFormatAuto, "2456572", reference, 24 * time.Hour, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199.120", reference, time.Microsecond, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199.120e3", reference, time.Microsecond, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199.120e6", reference, time.Microsecond, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199.120e9", reference, time.Microsecond, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199", reference, time.Second, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199120", reference, 0, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199120000", reference, 0, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199120000000", reference, 0, 0, false},
		{sqlite3.TimeFormatAuto, "2013-10-07 04:23:19.12-04:00", reference, 0, offset, false},
		{sqlite3.TimeFormatAuto, "04:23:19.12-04:00", refnodate, 0, offset, false},
		{sqlite3.TimeFormatAuto, "abc", time.Time{}, 0, 0, true},
		{sqlite3.TimeFormatAuto, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormat3, "2013-10-07 04:23:19.12-04:00", reference, 0, offset, false},
		{sqlite3.TimeFormat3, "2013-10-07 08:23:19.12", reference, 0, 0, false},
		{sqlite3.TimeFormat9, "04:23:19.12-04:00", refnodate, 0, offset, false},
		{sqlite3.TimeFormat9, "08:23:19.12", refnodate, 0, 0, false},
		{sqlite3.TimeFormat3, false, time.Time{}, 0, 0, true},
		{sqlite3.TimeFormat9, false, time.Time{}, 0, 0, true},

		{sqlite3.TimeFormatDefault, "2013-10-07T04:23:19.12-04:00", reference, 0, offset, false},
		{sqlite3.TimeFormatDefault, "2013-10-07T08:23:19.12Z", reference, 0, 0, false},
		{sqlite3.TimeFormatDefault, reference, reference, 0, offset, false},
		{sqlite3.TimeFormatDefault, false, time.Time{}, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := tt.fmt.Decode(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("%q.Decode(%v) error = %v, wantErr %v", tt.fmt, tt.val, err, tt.wantErr)
				return
			}
			if got.Sub(tt.want).Abs() > tt.wantDelta {
				t.Errorf("%q.Decode(%v) = %v, want %v", tt.fmt, tt.val, got, tt.want)
			}
			if _, off := got.Zone(); off != tt.wantOff {
				t.Errorf("%q.Decode(%v) = %v, want %v", tt.fmt, tt.val, off, tt.wantOff)
			}
		})
	}
}

func TestTimeFormat_Scanner(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(t.Context(), `CREATE TABLE test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))

	_, err = conn.ExecContext(t.Context(), `INSERT INTO test VALUES (?)`,
		sqlite3.TimeFormat7TZ.Encode(reference))
	if err != nil {
		t.Fatal(err)
	}

	var got time.Time
	err = conn.QueryRowContext(t.Context(), "SELECT * FROM test").
		Scan(sqlite3.TimeFormatAuto.Scanner(&got))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(reference) {
		t.Errorf("got %v, want %v", got, reference)
	}
}

func TestDB_timeCollation(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE times (tstamp COLLATE TIME)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`INSERT INTO times VALUES (?), (?), (?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	stmt.BindTime(1, time.Unix(0, 0).UTC(), sqlite3.TimeFormatDefault)
	stmt.BindTime(2, time.Unix(0, -1).UTC(), sqlite3.TimeFormatDefault)
	stmt.BindTime(3, time.Unix(0, +1).UTC(), sqlite3.TimeFormatDefault)
	stmt.Step()

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err = db.Prepare(`SELECT tstamp FROM times ORDER BY tstamp`)
	if err != nil {
		t.Fatal(err)
	}

	var t0 time.Time
	for stmt.Step() {
		t1 := stmt.ColumnTime(0, sqlite3.TimeFormatAuto)
		if t0.After(t1) {
			t.Errorf("got %v after %v", t0, t1)
		}
		t0 = t1
	}
	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDB_isoWeek(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT strftime('%G-W%V-%u', ?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	tend := time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC)
	tstart := time.Date(1500, 1, 1, 12, 0, 0, 0, time.UTC)
	for tm := tstart; tm.Before(tend); tm = tm.AddDate(0, 0, 1) {
		stmt.BindTime(1, tm, sqlite3.TimeFormatDefault)
		if !stmt.Step() {
			t.Fatal(stmt.Err())
		} else {
			y, w := tm.ISOWeek()
			d := tm.Weekday()
			if d == 0 {
				d = 7
			}
			want := fmt.Sprintf("%04d-W%02d-%d", y, w, d)
			if got := stmt.ColumnText(0); got != want {
				t.Errorf("got %q, want %q (%v)", got, want, tm)
			}
		}
		stmt.Reset()
	}
}
