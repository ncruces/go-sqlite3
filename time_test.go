package sqlite3

import (
	"reflect"
	"testing"
	"time"
)

func TestTimeFormat_Encode(t *testing.T) {
	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))

	tests := []struct {
		fmt  TimeFormat
		time time.Time
		want any
	}{
		{TimeFormatDefault, reference, "2013-10-07T04:23:19.12-04:00"},
		{TimeFormatJulianDay, reference, 2456572.849526851851852},
		{TimeFormatUnix, reference, int64(1381134199)},
		{TimeFormatUnixFrac, reference, 1381134199.120},
		{TimeFormatUnixMilli, reference, int64(1381134199_120)},
		{TimeFormatUnixMicro, reference, int64(1381134199_120000)},
		{TimeFormatUnixNano, reference, int64(1381134199_120000000)},
		{TimeFormat7, reference, "2013-10-07T08:23:19.120"},
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
	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))
	reftime := time.Date(2000, 1, 1, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))

	tests := []struct {
		fmt       TimeFormat
		val       any
		want      time.Time
		wantDelta time.Duration
		wantErr   bool
	}{
		{TimeFormatJulianDay, "2456572.849526851851852", reference, 0, false},
		{TimeFormatJulianDay, 2456572.849526851851852, reference, time.Millisecond, false},
		{TimeFormatJulianDay, int64(2456572), reference, 24 * time.Hour, false},
		{TimeFormatJulianDay, false, time.Time{}, 0, true},

		{TimeFormatUnix, "1381134199.120", reference, time.Microsecond, false},
		{TimeFormatUnix, 1381134199.120, reference, time.Microsecond, false},
		{TimeFormatUnix, int64(1381134199), reference, time.Second, false},
		{TimeFormatUnix, "abc", time.Time{}, 0, true},
		{TimeFormatUnix, false, time.Time{}, 0, true},

		{TimeFormatUnixMilli, "1381134199120", reference, 0, false},
		{TimeFormatUnixMilli, 1381134199.120e3, reference, 0, false},
		{TimeFormatUnixMilli, int64(1381134199_120), reference, 0, false},
		{TimeFormatUnixMilli, "abc", time.Time{}, 0, true},
		{TimeFormatUnixMilli, false, time.Time{}, 0, true},

		{TimeFormatUnixMicro, "1381134199120000", reference, 0, false},
		{TimeFormatUnixMicro, 1381134199.120e6, reference, 0, false},
		{TimeFormatUnixMicro, int64(1381134199_120000), reference, 0, false},
		{TimeFormatUnixMicro, "abc", time.Time{}, 0, true},
		{TimeFormatUnixMicro, false, time.Time{}, 0, true},

		{TimeFormatUnixNano, "1381134199120000000", reference, 0, false},
		{TimeFormatUnixNano, 1381134199.120e9, reference, 0, false},
		{TimeFormatUnixNano, int64(1381134199_120000000), reference, 0, false},
		{TimeFormatUnixNano, "abc", time.Time{}, 0, true},
		{TimeFormatUnixNano, false, time.Time{}, 0, true},

		{TimeFormatAuto, "2456572.849526851851852", reference, time.Millisecond, false},
		{TimeFormatAuto, "2456572", reference, 24 * time.Hour, false},
		{TimeFormatAuto, "1381134199.120", reference, time.Microsecond, false},
		{TimeFormatAuto, "1381134199.120e3", reference, time.Microsecond, false},
		{TimeFormatAuto, "1381134199.120e6", reference, time.Microsecond, false},
		{TimeFormatAuto, "1381134199.120e9", reference, time.Microsecond, false},
		{TimeFormatAuto, "1381134199", reference, time.Second, false},
		{TimeFormatAuto, "1381134199120", reference, 0, false},
		{TimeFormatAuto, "1381134199120000", reference, 0, false},
		{TimeFormatAuto, "1381134199120000000", reference, 0, false},
		{TimeFormatAuto, "2013-10-07 04:23:19.12-04:00", reference, 0, false},
		{TimeFormatAuto, "04:23:19.12-04:00", reftime, 0, false},
		{TimeFormatAuto, "abc", time.Time{}, 0, true},
		{TimeFormatAuto, false, time.Time{}, 0, true},

		{TimeFormat3, "2013-10-07 04:23:19.12-04:00", reference, 0, false},
		{TimeFormat3, "2013-10-07 08:23:19.12", reference, 0, false},
		{TimeFormat9, "04:23:19.12-04:00", reftime, 0, false},
		{TimeFormat9, "08:23:19.12", reftime, 0, false},
		{TimeFormat3, false, time.Time{}, 0, true},
		{TimeFormat9, false, time.Time{}, 0, true},

		{TimeFormatDefault, "2013-10-07T04:23:19.12-04:00", reference, 0, false},
		{TimeFormatDefault, "2013-10-07T08:23:19.12Z", reference, 0, false},
		{TimeFormatDefault, false, time.Time{}, 0, true},
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
