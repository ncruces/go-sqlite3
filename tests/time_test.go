package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
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

	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))
	reftime := time.Date(2000, 1, 1, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))

	tests := []struct {
		fmt       sqlite3.TimeFormat
		val       any
		want      time.Time
		wantDelta time.Duration
		wantErr   bool
	}{
		{sqlite3.TimeFormatJulianDay, "2456572.849526851851852", reference, 0, false},
		{sqlite3.TimeFormatJulianDay, 2456572.849526851851852, reference, time.Millisecond, false},
		{sqlite3.TimeFormatJulianDay, int64(2456572), reference, 24 * time.Hour, false},
		{sqlite3.TimeFormatJulianDay, false, time.Time{}, 0, true},

		{sqlite3.TimeFormatUnix, "1381134199.120", reference, time.Microsecond, false},
		{sqlite3.TimeFormatUnix, 1381134199.120, reference, time.Microsecond, false},
		{sqlite3.TimeFormatUnix, int64(1381134199), reference, time.Second, false},
		{sqlite3.TimeFormatUnix, "abc", time.Time{}, 0, true},
		{sqlite3.TimeFormatUnix, false, time.Time{}, 0, true},

		{sqlite3.TimeFormatUnixMilli, "1381134199120", reference, 0, false},
		{sqlite3.TimeFormatUnixMilli, 1381134199.120e3, reference, 0, false},
		{sqlite3.TimeFormatUnixMilli, int64(1381134199_120), reference, 0, false},
		{sqlite3.TimeFormatUnixMilli, "abc", time.Time{}, 0, true},
		{sqlite3.TimeFormatUnixMilli, false, time.Time{}, 0, true},

		{sqlite3.TimeFormatUnixMicro, "1381134199120000", reference, 0, false},
		{sqlite3.TimeFormatUnixMicro, 1381134199.120e6, reference, 0, false},
		{sqlite3.TimeFormatUnixMicro, int64(1381134199_120000), reference, 0, false},
		{sqlite3.TimeFormatUnixMicro, "abc", time.Time{}, 0, true},
		{sqlite3.TimeFormatUnixMicro, false, time.Time{}, 0, true},

		{sqlite3.TimeFormatUnixNano, "1381134199120000000", reference, 0, false},
		{sqlite3.TimeFormatUnixNano, 1381134199.120e9, reference, 0, false},
		{sqlite3.TimeFormatUnixNano, int64(1381134199_120000000), reference, 0, false},
		{sqlite3.TimeFormatUnixNano, "abc", time.Time{}, 0, true},
		{sqlite3.TimeFormatUnixNano, false, time.Time{}, 0, true},

		{sqlite3.TimeFormatAuto, "2456572.849526851851852", reference, time.Millisecond, false},
		{sqlite3.TimeFormatAuto, "2456572", reference, 24 * time.Hour, false},
		{sqlite3.TimeFormatAuto, "1381134199.120", reference, time.Microsecond, false},
		{sqlite3.TimeFormatAuto, "1381134199.120e3", reference, time.Microsecond, false},
		{sqlite3.TimeFormatAuto, "1381134199.120e6", reference, time.Microsecond, false},
		{sqlite3.TimeFormatAuto, "1381134199.120e9", reference, time.Microsecond, false},
		{sqlite3.TimeFormatAuto, "1381134199", reference, time.Second, false},
		{sqlite3.TimeFormatAuto, "1381134199120", reference, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199120000", reference, 0, false},
		{sqlite3.TimeFormatAuto, "1381134199120000000", reference, 0, false},
		{sqlite3.TimeFormatAuto, "2013-10-07 04:23:19.12-04:00", reference, 0, false},
		{sqlite3.TimeFormatAuto, "04:23:19.12-04:00", reftime, 0, false},
		{sqlite3.TimeFormatAuto, "abc", time.Time{}, 0, true},
		{sqlite3.TimeFormatAuto, false, time.Time{}, 0, true},

		{sqlite3.TimeFormat3, "2013-10-07 04:23:19.12-04:00", reference, 0, false},
		{sqlite3.TimeFormat3, "2013-10-07 08:23:19.12", reference, 0, false},
		{sqlite3.TimeFormat9, "04:23:19.12-04:00", reftime, 0, false},
		{sqlite3.TimeFormat9, "08:23:19.12", reftime, 0, false},
		{sqlite3.TimeFormat3, false, time.Time{}, 0, true},
		{sqlite3.TimeFormat9, false, time.Time{}, 0, true},

		{sqlite3.TimeFormatDefault, "2013-10-07T04:23:19.12-04:00", reference, 0, false},
		{sqlite3.TimeFormatDefault, "2013-10-07T08:23:19.12Z", reference, 0, false},
		{sqlite3.TimeFormatDefault, false, time.Time{}, 0, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := tt.fmt.Decode(tt.val)
			if (err != nil) != tt.wantErr {
				t.Errorf("%q.Decode(%v) error = %v, wantErr %v", tt.fmt, tt.val, err, tt.wantErr)
				return
			}
			if tt.want.Sub(got).Abs() > tt.wantDelta {
				t.Errorf("%q.Decode(%v) = %v, want %v", tt.fmt, tt.val, got, tt.want)
			}
		})
	}
}
