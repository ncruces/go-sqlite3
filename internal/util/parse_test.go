package util_test

import (
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		str string
		val bool
		ok  bool
	}{
		{"", false, false},
		{"0", false, true},
		{"1", true, true},
		{"9", true, true},
		{"T", false, false},
		{"true", true, true},
		{"FALSE", false, true},
		{"false?", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			gotVal, gotOK := util.ParseBool(tt.str)
			if gotVal != tt.val || gotOK != tt.ok {
				t.Errorf("ParseBool(%q) = (%v, %v) want (%v, %v)", tt.str, gotVal, gotOK, tt.val, tt.ok)
			}
		})
	}
}

func TestParseTimeShift(t *testing.T) {
	epoch := time.Unix(0, 0)
	tests := []struct {
		str string
		val time.Time
		ok  bool
	}{
		{"", epoch, false},
		{"0001-11-30", epoch, false},
		{"+_001-11-30", epoch, false},
		{"+0001-_1-30", epoch.AddDate(1, 0, 0), false},
		{"+0001-11-_0", epoch.AddDate(1, 11, 0), false},
		{"+0001-11-30", epoch.AddDate(1, 11, 30), true},
		{"-0001-11-30", epoch.AddDate(-1, -11, -30), true},
		{"+0001-11-30T", epoch.AddDate(1, 11, 30), false},
		{"+0001-11-30 12", epoch.AddDate(1, 11, 30), false},
		{"+0001-11-30 _2:30", epoch.AddDate(1, 11, 30), false},
		{"+0001-11-30 12:_0", epoch.AddDate(1, 11, 30), false},
		{"+0001-11-30 12:30", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute), true},
		{"+0001-11-30 12:30:", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute), false},
		{"+0001-11-30 12:30:_0", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute), false},
		{"+0001-11-30 12:30:59", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute + 59*time.Second), true},
		{"+0001-11-30 12:30:59.", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute + 59*time.Second), false},
		{"+0001-11-30 12:30:59._", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute + 59*time.Second), false},
		{"+0001-11-30 12:30:59.1", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute + 59*time.Second + 100*time.Millisecond), true},
		{"+0001-11-30 12:30:59.123456789_", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute + 59*time.Second + 123456789), false},
		{"+0001-11-30 12:30:59.1234567890", epoch.AddDate(1, 11, 30).Add(12*time.Hour + 30*time.Minute + 59*time.Second + 123456789), true},
		{"-12:30:59.1234567890", epoch.Add(-12*time.Hour - 30*time.Minute - 59*time.Second - 123456789), true},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			years, months, days, duration, gotOK := util.ParseTimeShift(tt.str)
			gotVal := epoch.AddDate(years, months, days).Add(duration)
			if !gotVal.Equal(tt.val) || gotOK != tt.ok {
				t.Errorf("ParseTimeShift(%q) = (%v, %v) want (%v, %v)", tt.str, gotVal, gotOK, tt.val, tt.ok)
			}
		})
	}
}

func FuzzParseTimeShift(f *testing.F) {
	f.Add("")
	f.Add("0001-12-30")
	f.Add("+_001-12-30")
	f.Add("+0001-_2-30")
	f.Add("+0001-12-_0")
	f.Add("+0001-12-30")
	f.Add("-0001-12-30")
	f.Add("+0001-12-30T")
	f.Add("+0001-12-30 12")
	f.Add("+0001-12-30 _2:30")
	f.Add("+0001-12-30 12:_0")
	f.Add("+0001-12-30 12:30")
	f.Add("+0001-12-30 12:30:")
	f.Add("+0001-12-30 12:30:_0")
	f.Add("+0001-12-30 12:30:60")
	f.Add("+0001-12-30 12:30:60.")
	f.Add("+0001-12-30 12:30:60._")
	f.Add("+0001-12-30 12:30:60.1")
	f.Add("+0001-12-30 12:30:60.123456789_")
	f.Add("+0001-12-30 12:30:60.1234567890")
	f.Add("-12:30:60.1234567890")

	c, err := sqlite3.Open(":memory:")
	if err != nil {
		f.Fatal(err)
	}
	defer c.Close()

	s, _, err := c.Prepare(`SELECT julianday('00:00', ?)`)
	if err != nil {
		f.Fatal(err)
	}
	defer s.Close()

	// Default SQLite date.
	epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, str string) {
		years, months, days, duration, ok := util.ParseTimeShift(str)

		// Account for a full 400 year cycle.
		if years < -200 || years > +200 {
			t.Skip()
		}
		// SQLite only tracks milliseconds.
		if duration != duration.Truncate(time.Millisecond) {
			t.Skip()
		}

		if ok {
			s.Reset()
			s.BindText(1, str)
			if !s.Step() {
				t.Fail()
			}

			got := epoch.AddDate(years, months, days).Add(duration)

			// Julian day introduces floating point inaccuracy.
			want := s.ColumnTime(0, sqlite3.TimeFormatJulianDay)
			want = want.Round(time.Millisecond)
			if !got.Equal(want) {
				t.Fatalf("with %q, got %v, want %v", str, got, want)
			}
		}
	})
}
