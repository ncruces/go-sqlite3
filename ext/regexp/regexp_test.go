package regexp

import (
	"database/sql"
	"regexp"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestRegister(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, Register)
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
		{`regexp_replace('Hello', 'llo', 'll')`, "Hell"},
		// https://postgresql.org/docs/current/functions-matching.html
		{`regexp_count('ABCABCAXYaxy', 'A.')`, "3"},
		{`regexp_count('ABCABCAXYaxy', '(?i)A.', 1)`, "4"},
		{`regexp_instr('number of your street, town zip, FR', '[^,]+', 1, 2)`, "23"},
		{`regexp_instr('ABCDEFGHI', '(?i)(c..)(...)', 1, 1, 0, 2)`, "6"},
		{`regexp_substr('number of your street, town zip, FR', '[^,]+', 1, 2)`, " town zip"},
		{`regexp_substr('ABCDEFGHI', '(?i)(c..)(...)', 1, 1, 2)`, "FGH"},
		{`regexp_replace('foobarbaz', 'b..', 'X', 1, 1)`, "fooXbaz"},
		{`regexp_replace('foobarbaz', 'b..', 'X')`, "fooXX"},
		{`regexp_replace('foobarbaz', 'b(..)', 'X${1}Y')`, "fooXarYXazY"},
		{`regexp_replace('A PostgreSQL function', '(?i)a|e|i|o|u', 'X', 1, 0)`, "X PXstgrXSQL fXnctXXn"},
		{`regexp_replace('A PostgreSQL function', '(?i)a|e|i|o|u', 'X', 1, 3)`, "A PostgrXSQL function"},
		// https://docs.oracle.com/en/database/oracle/oracle-database/21/sqlrf/REGEXP_COUNT.html
		{`regexp_count('123123123123123', '(12)3', 1)`, "5"},
		{`regexp_count('123123123123', '123', 3)`, "3"},
		{`regexp_instr('500 Oracle Parkway, Redwood Shores, CA', '[^ ]+', 1, 6)`, "37"},
		{`regexp_instr('500 Oracle Parkway, Redwood Shores, CA', '(?i)[s|r|p][[:alpha:]]{6}', 3, 2, 1)`, "28"},
		{`regexp_instr('1234567890', '(123)(4(56)(78))', 1, 1, 0, 1)`, "1"},
		{`regexp_instr('1234567890', '(123)(4(56)(78))', 1, 1, 0, 2)`, "4"},
		{`regexp_instr('1234567890', '(123)(4(56)(78))', 1, 1, 0, 4)`, "7"},
		{`regexp_substr('500 Oracle Parkway, Redwood Shores, CA', ',[^,]+,')`, ", Redwood Shores,"},
		{`regexp_substr('http://www.example.com/products', 'http://([[:alnum:]]+\.?){3,4}/?')`, "http://www.example.com/"},
		{`regexp_substr('1234567890', '(123)(4(56)(78))', 1, 1, 1)`, "123"},
		{`regexp_substr('1234567890', '(123)(4(56)(78))', 1, 1, 4)`, "78"},
		{`regexp_substr('123123123123', '1(.)3', 3, 2, 1)`, "2"},
		{`regexp_replace('500   Oracle     Parkway,    Redwood  Shores, CA', '( ){2,}', ' ')`, "500 Oracle Parkway, Redwood Shores, CA"},
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
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, Register)
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

func TestRegister_pointer(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got int
	err = db.QueryRow(`SELECT regexp_count('ABCABCAXYaxy', ?, 1)`,
		sqlite3.Pointer(regexp.MustCompile(`(?i)A.`))).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 4 {
		t.Errorf("got %d, want %d", got, 4)
	}
}

func TestGlobPrefix(t *testing.T) {
	tests := []struct {
		re   string
		want string
	}{
		{`[`, ""},
		{``, "*"},
		{`^`, "*"},
		{`a`, "*"},
		{`ab`, "*"},
		{`^a`, "a*"},
		{`^a*`, "*"},
		{`^a+`, "a*"},
		{`^ab*`, "a*"},
		{`^ab+`, "ab*"},
		{`^a\?b`, "a*"},
		{`^[a-z]`, "*"},
	}
	for _, tt := range tests {
		t.Run(tt.re, func(t *testing.T) {
			if got := GlobPrefix(tt.re); got != tt.want {
				t.Errorf("GlobPrefix(%v) = %v, want %v", tt.re, got, tt.want)
			}
		})
	}
}

func FuzzGlobPrefix(f *testing.F) {
	f.Add(``, ``)
	f.Add(`[`, ``)
	f.Add(`^`, ``)
	f.Add(`a`, `a`)
	f.Add(`ab`, `b`)
	f.Add(`^a`, `a`)
	f.Add(`^a*`, `ab`)
	f.Add(`^a+`, `ab`)
	f.Add(`^ab*`, `ab`)
	f.Add(`^ab+`, `ab`)
	f.Add(`^a\?b`, `ab`)
	f.Add(`^[a-z]`, `ab`)

	f.Fuzz(func(t *testing.T, lit, str string) {
		re, err := regexp.Compile(lit)
		if err != nil {
			t.SkipNow()
		}
		if re.MatchString(str) {
			prefix, ok := strings.CutSuffix(GlobPrefix(lit), "*")
			if !ok {
				t.Fatalf("missing * after %q for %q with %q", prefix, lit, str)
			}
			if !strings.HasPrefix(str, prefix) {
				t.Fatalf("missing prefix %q for %q with %q", prefix, lit, str)
			}
		}
	})
}
