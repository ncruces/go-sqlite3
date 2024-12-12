package driver

import (
	"context"
	"testing"
)

func Fuzz_isWhitespace(f *testing.F) {
	f.Add("")
	f.Add(" ")
	f.Add(";")
	f.Add("0")
	f.Add("-")
	f.Add("--")
	f.Add("/*")
	f.Add("/*/")
	f.Add("/**")
	f.Add("/**0/")
	f.Add("\v")
	f.Add(" \v")
	f.Add("\xf0")

	db, err := Open(":memory:")
	if err != nil {
		f.Fatal(err)
	}
	defer db.Close()

	f.Fuzz(func(t *testing.T, str string) {
		c, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}

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
