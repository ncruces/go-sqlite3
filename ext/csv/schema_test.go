package csv

import "testing"

func Test_getSchema(t *testing.T) {
	tests := []struct {
		header  bool
		columns int
		row     []string
		want    string
	}{
		{true, 2, nil, `CREATE TABLE x(c1 TEXT,c2 TEXT)`},
		{false, 2, nil, `CREATE TABLE x(c1 TEXT,c2 TEXT)`},
		{false, -1, []string{"abc", ""}, `CREATE TABLE x(c1 TEXT,c2 TEXT)`},
		{true, 3, []string{"abc", ""}, `CREATE TABLE x("abc" TEXT,c2 TEXT,c3 TEXT)`},
		{true, -1, []string{"abc", "def"}, `CREATE TABLE x("abc" TEXT,"def" TEXT)`},
		{true, 1, []string{"abc", "def"}, `CREATE TABLE x("abc" TEXT)`},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := getSchema(tt.header, tt.columns, tt.row); got != tt.want {
				t.Errorf("getSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}
