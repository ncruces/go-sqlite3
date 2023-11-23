package sqlite3

import "testing"

func Test_emptyStatement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		stmt string
		want bool
	}{
		{"empty", "", true},
		{"space", " ", true},
		{"separator", ";\n ", true},
		{"begin", "BEGIN", false},
		{"select", "SELECT 1;", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := emptyStatement(tt.stmt); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func Fuzz_emptyStatement(f *testing.F) {
	f.Add("")
	f.Add(" ")
	f.Add(";\n ")
	f.Add("; ;\v")
	f.Add("BEGIN")
	f.Add("SELECT 1;")

	db, err := Open(":memory:")
	if err != nil {
		f.Fatal(err)
	}
	defer db.Close()

	f.Fuzz(func(t *testing.T, sql string) {
		// If empty, SQLite parses it as empty.
		if emptyStatement(sql) {
			stmt, tail, err := db.Prepare(sql)
			if err != nil {
				t.Errorf("%q, %v", sql, err)
			}
			if stmt != nil {
				t.Errorf("%q, %v", sql, stmt)
			}
			if tail != "" {
				t.Errorf("%q", sql)
			}
			stmt.Close()
		}
	})
}
