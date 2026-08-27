package fts5_test

import (
	"errors"
	"fmt"
	"log"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
	"golang.org/x/text/cases"
)

func Example() {
	db, err := driver.Open("file:/test.db?vfs=memdb", func(c *sqlite3.Conn) error {
		return fts5.RegisterCustom(c, func(a *fts5.API) error {
			return errors.Join(
				a.CreateTokenizer("utf8", func(arg []string) (fts5.Tokenizer, error) {
					return Utf8Tokenizer{}, nil
				}),
				a.CreateFunction("hit_count", func(fts fts5.Context, ctx sqlite3.Context, arg ...sqlite3.Value) {
					if count, err := fts.InstCount(); err != nil {
						ctx.ResultError(err)
					} else {
						ctx.ResultInt(count)
					}
				}))
		})
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE docs USING fts5(title, body, tokenize=utf8);
		INSERT INTO docs(title, body) VALUES 
			('Go Programming', 'An intensive guide to Go routines.'),
			('SQLite Tutorial', 'Learn how to use virtual tables efficiently.');
	`)
	if err != nil {
		log.Fatal(err)
	}

	var title string
	var hits int
	err = db.QueryRow("SELECT title, hit_count(docs) FROM docs WHERE docs MATCH 'go AND routines'").Scan(&title, &hits)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s: %d hits\n", title, hits)
	// Output: Go Programming: 3 hits
}

type Utf8Tokenizer struct{}

func (Utf8Tokenizer) Tokenize(flags fts5.TokenizeFlag, text, locale string, token fts5.TokenCallback) error {
	folder := cases.Fold()
	isToken := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.Is(unicode.Co, r)
	}

	var start int
	for start < len(text) {
		for start < len(text) {
			r, sz := utf8.DecodeRuneInString(text[start:])
			if isToken(r) {
				break
			}
			start += sz
		}

		end := start
		for end < len(text) {
			r, sz := utf8.DecodeRuneInString(text[end:])
			if !isToken(r) {
				break
			}
			end += sz
		}

		if start < end {
			err := token(0, folder.String(text[start:end]), start, end)
			if err != nil {
				return err
			}
			start = end
		}
	}
	return nil
}

func TestContext(t *testing.T) {
	db, err := driver.Open("file:/test_context.db?vfs=memdb", func(c *sqlite3.Conn) error {
		return fts5.RegisterCustom(c, func(a *fts5.API) error {
			return a.CreateFunction("test", func(fts fts5.Context, ctx sqlite3.Context, arg ...sqlite3.Value) {
				if v := fts.ColumnCount(); v != 2 {
					t.Errorf("ColumnCount: got %d, want 2", v)
				}
				if v, err := fts.RowCount(); err != nil || v != 2 {
					t.Errorf("RowCount: got %d err=%v, want 2", v, err)
				}
				if v, err := fts.ColumnTotalSize(0); err != nil || v != 4 {
					t.Errorf("ColumnTotalSize(0): got %d err=%v, want 4", v, err)
				}
				if v, err := fts.ColumnTotalSize(1); err != nil || v != 13 {
					t.Errorf("ColumnTotalSize(1): got %d err=%v, want 13", v, err)
				}
				if v := fts.PhraseCount(); v != 2 {
					t.Errorf("PhraseCount: got %d, want 2", v)
				}
				if v := fts.PhraseSize(0); v != 1 {
					t.Errorf("PhraseSize(0): got %d, want 1", v)
				}

				want := []struct{ p, c, o int }{{0, 0, 0}, {0, 1, 4}, {1, 1, 5}}
				if v, err := fts.InstCount(); err != nil || v != len(want) {
					t.Errorf("InstCount: got %d, want %d", v, len(want))
				}
				for i := range want {
					p, c, o, err := fts.Inst(i)
					if err != nil || p != want[i].p || c != want[i].c || o != want[i].o {
						t.Errorf("Inst(%d): got %d,%d,%d err=%v, want %v", i, p, c, o, err, want[i])
					}
				}

				if v, err := fts.InstToken(0, 0); err != nil || v != "go" {
					t.Errorf("InstToken(0,0): got %q err=%v, want 'go'", v, err)
				}
				if v, err := fts.InstToken(2, 0); err != nil || v != "routines" {
					t.Errorf("InstToken(2,0): got %q err=%v, want 'routines'", v, err)
				}
				if v := fts.RowID(); v != 1 {
					t.Errorf("RowID: got %d, want 1", v)
				}
				if v, err := fts.ColumnText(0); err != nil || v != "Go Programming" {
					t.Errorf("ColumnText(0): got %q err=%v, want 'Go Programming'", v, err)
				}
				if v, err := fts.ColumnSize(0); err != nil || v != 2 {
					t.Errorf("ColumnSize(0): got %d err=%v, want 2", v, err)
				}
				if v, err := fts.ColumnSize(1); err != nil || v != 6 {
					t.Errorf("ColumnSize(1): got %d err=%v, want 6", v, err)
				}
				if err := fts.SetAuxdata("aux"); err != nil {
					t.Errorf("SetAuxdata: %v", err)
				}
				if v := fts.GetAuxdata(false); v != "aux" {
					t.Errorf("GetAuxdata: got %v, want 'aux'", v)
				}
				if v, err := fts.QueryToken(0, 0); err != nil || v != "go" {
					t.Errorf("QueryToken(0,0): got %q err=%v, want 'go'", v, err)
				}
				if v, err := fts.QueryToken(1, 0); err != nil || v != "routines" {
					t.Errorf("QueryToken(1,0): got %q err=%v, want 'routines'", v, err)
				}
				if v, err := fts.ColumnLocale(0); err != nil || v != "" {
					t.Errorf("ColumnLocale(0): got %q err=%v, want ''", v, err)
				}
			})
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE docs USING fts5(title, body, tokenize=ascii);
		INSERT INTO docs(title, body) VALUES 
			('Go Programming', 'An intensive guide to Go routines.'),
			('SQLite Tutorial', 'Learn how to use virtual tables efficiently.');
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("SELECT title, test(docs) FROM docs WHERE docs MATCH 'go AND routines'")
	if err != nil {
		t.Fatal(err)
	}
}
