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

func TestGetAffinity(t *testing.T) {
	tests := []struct {
		decl string
		want sql3util.Affinity
	}{
		{"", sql3util.BLOB},
		{"INTEGER", sql3util.INTEGER},
		{"TINYINT", sql3util.INTEGER},
		{"TEXT", sql3util.TEXT},
		{"CHAR", sql3util.TEXT},
		{"CLOB", sql3util.TEXT},
		{"BLOB", sql3util.BLOB},
		{"REAL", sql3util.REAL},
		{"FLOAT", sql3util.REAL},
		{"DOUBLE", sql3util.REAL},
		{"NUMERIC", sql3util.NUMERIC},
		{"DECIMAL", sql3util.NUMERIC},
		{"BOOLEAN", sql3util.NUMERIC},
		{"DATETIME", sql3util.NUMERIC},
	}
	for _, tt := range tests {
		t.Run(tt.decl, func(t *testing.T) {
			if got := sql3util.GetAffinity(tt.decl); got != tt.want {
				t.Errorf("GetAffinity() = %v, want %v", got, tt.want)
			}
		})
	}
}
