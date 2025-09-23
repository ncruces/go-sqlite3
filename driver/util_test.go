package driver

import (
	"database/sql/driver"
	"slices"
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func Test_namedValues(t *testing.T) {
	want := []driver.NamedValue{
		{Ordinal: 1, Value: true},
		{Ordinal: 2, Value: false},
	}
	got := namedValues([]driver.Value{true, false})
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Fuzz_notWhitespace(f *testing.F) {
	f.Add("")
	f.Add(" ")
	f.Add(";")
	f.Add("0")
	f.Add("-")
	f.Add("-0")
	f.Add("--")
	f.Add("--0")
	f.Add("--\n")
	f.Add("--0\n")
	f.Add("/0")
	f.Add("/*")
	f.Add("/*/")
	f.Add("/**")
	f.Add("/*0")
	f.Add("/**/")
	f.Add("/***/")
	f.Add("/**0/")
	f.Add("\v")
	f.Add(" \v")
	f.Add("\xf0")
	f.Add("\000")

	db, err := Open(":memory:")
	if err != nil {
		f.Fatal(err)
	}
	defer db.Close()

	f.Fuzz(func(t *testing.T, str string) {
		if len(str) > 128 {
			t.SkipNow()
		}

		c, err := db.Conn(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		c.Raw(func(driverConn any) error {
			conn := driverConn.(*conn).Conn
			stmt, tail, err := conn.Prepare(str)
			stmt.Close()

			// It's hard to be bug for bug compatible with SQLite.
			// We settle for somewhat less:
			//   - if SQLite reports whitespace, we must too
			//   - if we report whitespace, SQLite must not parse a statement
			if notWhitespace(str) {
				if stmt == nil && tail == "" && err == nil {
					t.Errorf("was whitespace: %q", str)
				}
			} else {
				if stmt != nil {
					t.Errorf("was not whitespace: %q (%v)", str, err)
				}
			}
			return nil
		})
	})
}
