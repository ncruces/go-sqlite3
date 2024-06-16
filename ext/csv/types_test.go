package csv

import (
	_ "embed"
	"testing"
)

func Test_getAffinity(t *testing.T) {
	tests := []struct {
		decl string
		want affinity
	}{
		{"", blob},
		{"INTEGER", integer},
		{"TINYINT", integer},
		{"TEXT", text},
		{"CHAR", text},
		{"CLOB", text},
		{"BLOB", blob},
		{"REAL", real},
		{"FLOAT", real},
		{"DOUBLE", real},
		{"NUMERIC", numeric},
		{"DECIMAL", numeric},
		{"BOOLEAN", numeric},
		{"DATETIME", numeric},
	}
	for _, tt := range tests {
		t.Run(tt.decl, func(t *testing.T) {
			if got := getAffinity(tt.decl); got != tt.want {
				t.Errorf("getAffinity() = %v, want %v", got, tt.want)
			}
		})
	}
}
