package sql3util_test

import (
	"testing"

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
