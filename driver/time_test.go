package driver

import (
	"testing"
	"time"
)

// This checks that any string can be recovered as the same string.
func Fuzz_stringOrTime_1(f *testing.F) {
	f.Add("")
	f.Add(" ")
	f.Add("SQLite")
	f.Add(time.RFC3339)
	f.Add(time.RFC3339Nano)
	f.Add(time.Layout)
	f.Add(time.DateTime)
	f.Add(time.DateOnly)
	f.Add(time.TimeOnly)
	f.Add("2006-01-02T15:04:05Z")
	f.Add("2006-01-02T15:04:05.000Z")
	f.Add("2006-01-02T15:04:05.9999999999Z")
	f.Add("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.")

	f.Fuzz(func(t *testing.T, str string) {
		v, ok := maybeTime("", str)
		if ok {
			// Make sure times round-trip to the same string:
			// https://pkg.go.dev/database/sql#Rows.Scan
			if v.Format(time.RFC3339Nano) != str {
				t.Fatalf("did not round-trip: %q", str)
			}
		} else {
			date, err := time.Parse(time.RFC3339Nano, str)
			if err == nil && date.Format(time.RFC3339Nano) == str {
				t.Fatalf("would round-trip: %q", str)
			}
		}
	})
}

// This checks that any [time.Time] can be recovered as a [time.Time],
// with nanosecond accuracy, and preserving any timezone offset.
func Fuzz_stringOrTime_2(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(0), int64(1))
	f.Add(int64(0), int64(-1))
	f.Add(int64(0), int64(999_999_999))
	f.Add(int64(0), int64(1_000_000_000))
	f.Add(int64(7956915742), int64(222_222_222))    // twosday
	f.Add(int64(639095955742), int64(222_222_222))  // twosday, year 22222AD
	f.Add(int64(-763421161058), int64(222_222_222)) // twosday, year 22222BC

	checkTime := func(t testing.TB, date time.Time) {
		v, ok := maybeTime("", date.Format(time.RFC3339Nano))
		if ok {
			// Make sure times round-trip to the same time:
			if !v.Equal(date) {
				t.Fatalf("did not round-trip: %v", date)
			}
			// With the same zone offset:
			_, off1 := v.Zone()
			_, off2 := date.Zone()
			if off1 != off2 {
				t.Fatalf("did not round-trip: %v", date)
			}
		} else {
			t.Fatalf("was not recovered: %v", date)
		}
	}

	f.Fuzz(func(t *testing.T, sec, nsec int64) {
		// Reduce the search space.
		if 1e12 < sec || sec < -1e12 {
			// Dates before 29000BC and after 33000AD; I think we're safe.
			return
		}
		if 0 < nsec || nsec > 1e10 {
			// Out of range nsec: [time.Time.Unix] handles these.
			return
		}

		unix := time.Unix(sec, nsec)
		checkTime(t, unix)
		checkTime(t, unix.UTC())
		checkTime(t, unix.In(time.FixedZone("", -8*3600)))
		checkTime(t, unix.In(time.FixedZone("", +8*3600)))
	})
}
