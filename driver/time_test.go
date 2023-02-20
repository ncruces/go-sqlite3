package driver

import (
	"testing"
	"time"
)

func Fuzz_maybeDate(f *testing.F) {
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
		value := maybeDate(str)

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
