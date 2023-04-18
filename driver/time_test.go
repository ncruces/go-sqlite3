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
		value := stringOrTime([]byte(str))

		switch v := value.(type) {
		case time.Time:
			// Make sure times round-trip to the same string:
			// https://pkg.go.dev/database/sql#Rows.Scan
			if v.Format(time.RFC3339Nano) != str {
				t.Fatalf("did not round-trip: %q", str)
			}
		case string:
			if v != str {
				t.Fatalf("did not round-trip: %q", str)
			}

			date, err := time.Parse(time.RFC3339Nano, str)
			if err == nil && date.Format(time.RFC3339Nano) == str {
				t.Fatalf("would round-trip: %q", str)
			}
		default:
			t.Fatalf("invalid type %T: %q", v, str)
		}
	})
}

// This checks that any [time.Time] can be recovered as a [time.Time],
// with nanosecond accuracy, and preserving any timezone offset.
func Fuzz_stringOrTime_2(f *testing.F) {
	f.Add(0, 0)
	f.Add(0, 1)
	f.Add(0, -1)
	f.Add(0, 999_999_999)
	f.Add(0, 1_000_000_000)
	f.Add(7956915742, 222_222_222)    // twosday
	f.Add(639095955742, 222_222_222)  // twosday, year 22222AD
	f.Add(-763421161058, 222_222_222) // twosday, year 22222BC

	checkTime := func(t *testing.T, date time.Time) {
		value := stringOrTime([]byte(date.Format(time.RFC3339Nano)))

		switch v := value.(type) {
		case time.Time:
			// Make sure times round-trip to the same time:
			if !v.Equal(date) {
				t.Fatalf("did not round-trip: %v", date)
			}
			// Make with the same zone offset:
			_, off1 := v.Zone()
			_, off2 := date.Zone()
			if off1 != off2 {
				t.Fatalf("did not round-trip: %v", date)
			}
		case string:
			t.Fatalf("was not recovered: %v", date)
		default:
			t.Fatalf("invalid type %T: %v", v, date)
		}
	}

	f.Fuzz(func(t *testing.T, sec, nsec int) {
		// Reduce the search space.
		if 1e12 < sec || sec < -1e12 {
			// Dates before 29000BC and after 33000AD; I think we're safe.
			return
		}
		if 0 < nsec || nsec > 1e10 {
			// Out of range nsec: [time.Time.Unix] handles these.
			return
		}

		unix := time.Unix(int64(sec), int64(nsec))
		checkTime(t, unix)
		checkTime(t, unix.UTC())
		checkTime(t, unix.In(time.FixedZone("", -8*3600)))
		checkTime(t, unix.In(time.FixedZone("", +8*3600)))
	})
}
