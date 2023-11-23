package csv

import "testing"

func Test_getSchema(t *testing.T) {
	tests := []struct {
		header  bool
		columns int
		row     []string
		want    string
	}{
		{true, 2, nil, `CREATE TABLE x(c1,c2)`},
		{false, 2, nil, `CREATE TABLE x(c1,c2)`},
		{true, 3, []string{"abc", ""}, `CREATE TABLE x("abc",c2,c3)`},
		{true, 1, []string{"abc", "def"}, `CREATE TABLE x("abc")`},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := getSchema(tt.header, tt.columns, tt.row); got != tt.want {
				t.Errorf("getSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}
