package tests

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func TestDatatype_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data sqlite3.Datatype
		want string
	}{
		{sqlite3.INTEGER, "INTEGER"},
		{sqlite3.FLOAT, "FLOAT"},
		{sqlite3.TEXT, "TEXT"},
		{sqlite3.BLOB, "BLOB"},
		{sqlite3.NULL, "NULL"},
		{10, "10"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.data.String(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
