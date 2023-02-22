package sqlite3

import "testing"

func TestDatatype_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data Datatype
		want string
	}{
		{INTEGER, "INTEGER"},
		{FLOAT, "FLOAT"},
		{TEXT, "TEXT"},
		{BLOB, "BLOB"},
		{NULL, "NULL"},
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
