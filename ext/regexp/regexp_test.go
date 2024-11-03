package regexp

import (
	"database/sql"
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestRegister(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		test string
		want string
	}{
		{`'Hello' REGEXP 'elo'`, "0"},
		{`'Hello' REGEXP 'ell'`, "1"},
		{`'Hello' REGEXP 'el.'`, "1"},
		{`regexp_like('Hello', 'elo')`, "0"},
		{`regexp_like('Hello', 'ell')`, "1"},
		{`regexp_like('Hello', 'el.')`, "1"},
		{`regexp_count('Hello', 'l')`, "2"},
		{`regexp_instr('Hello', 'el.')`, "2"},
		{`regexp_instr('Hello', '.', 6)`, ""},
		{`regexp_substr('Hello', 'el.')`, "ell"},
		{`regexp_substr('Hello', 'l', 2, 2)`, "l"},
		{`regexp_replace('Hello', 'llo', 'll')`, "Hell"},

		{`regexp_count('123123123123123', '(12)3', 1)`, "5"},
		{`regexp_instr('500 Oracle Parkway, Redwood Shores, CA', '(?i)[s|r|p][[:alpha:]]{6}', 3, 2, 1)`, "28"},
		{`regexp_substr('500 Oracle Parkway, Redwood Shores, CA', ',[^,]+,', 3, 1)`, ", Redwood Shores,"},
		{`regexp_replace('500   Oracle     Parkway,    Redwood  Shores, CA', '( ){2,}', ' ', 3)`, "500 Oracle Parkway, Redwood Shores, CA"},
	}

	for _, tt := range tests {
		var got sql.NullString
		err := db.QueryRow(`SELECT ` + tt.test).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got.String != tt.want {
			t.Errorf("got %q, want %q", got.String, tt.want)
		}
	}
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []string{
		`'' REGEXP ?`,
		`regexp_like('', ?)`,
		`regexp_count('', ?)`,
		`regexp_instr('', ?)`,
		`regexp_substr('', ?)`,
		`regexp_replace('', ?, '')`,
	}

	for _, tt := range tests {
		err := db.QueryRow(`SELECT `+tt, `\`).Scan(nil)
		if err == nil {
			t.Fatal("want error")
		}
	}
}
