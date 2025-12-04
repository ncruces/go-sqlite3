package sql3util_test

import (
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func TestUnquote(t *testing.T) {
	tests := []struct {
		val  string
		want string
	}{
		{"a", "a"},
		{"abc", "abc"},
		{"abba", "abba"},
		{"`ab``c`", "ab`c"},
		{"'ab''c'", "ab'c"},
		{"'ab``c'", "ab``c"},
		{"[ab``c]", "ab``c"},
		{`"ab""c"`, `ab"c`},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if got := sql3util.Unquote(tt.val); got != tt.want {
				t.Errorf("Unquote(%s) = %s, want %s", tt.val, got, tt.want)
			}
		})
	}
}

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
			gotVal, gotOK := sql3util.ParseBool(tt.str)
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
		{"0001-12-30", epoch, false},
		{"+_001-12-30", epoch, false},
		{"+0001-_2-30", epoch.AddDate(1, 0, 0), false},
		{"+0001-12-_0", epoch.AddDate(1, 12, 0), false},
		{"+0001-12-30", epoch.AddDate(1, 12, 30), true},
		{"-0001-12-30", epoch.AddDate(-1, -12, -30), true},
		{"+0001-12-30T", epoch.AddDate(1, 12, 30), false},
		{"+0001-12-30 12", epoch.AddDate(1, 12, 30), false},
		{"+0001-12-30 _2:30", epoch.AddDate(1, 12, 30), false},
		{"+0001-12-30 12:_0", epoch.AddDate(1, 12, 30), false},
		{"+0001-12-30 12:30", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 30*time.Minute), true},
		{"+0001-12-30 12:30:", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 30*time.Minute), false},
		{"+0001-12-30 12:30:_0", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 30*time.Minute), false},
		{"+0001-12-30 12:30:60", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 31*time.Minute), true},
		{"+0001-12-30 12:30:60.", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 31*time.Minute), false},
		{"+0001-12-30 12:30:60._", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 31*time.Minute), false},
		{"+0001-12-30 12:30:60.1", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 31*time.Minute + 100*time.Millisecond), true},
		{"+0001-12-30 12:30:60.123456789_", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 31*time.Minute + 123456789), false},
		{"+0001-12-30 12:30:60.1234567890", epoch.AddDate(1, 12, 30).Add(12*time.Hour + 31*time.Minute + 123456789), true},
		{"-12:30:60.1234567890", epoch.Add(-12*time.Hour - 31*time.Minute - 123456789), true},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			years, months, days, duration, gotOK := sql3util.ParseTimeShift(tt.str)
			gotVal := epoch.AddDate(years, months, days).Add(duration)
			if !gotVal.Equal(tt.val) || gotOK != tt.ok {
				t.Errorf("ParseTimeShift(%q) = (%v, %v) want (%v, %v)", tt.str, gotVal, gotOK, tt.val, tt.ok)
			}
		})
	}
}
