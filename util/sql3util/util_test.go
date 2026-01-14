package sql3util_test

import (
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
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
