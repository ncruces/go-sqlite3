package fts5_test

import (
	"fmt"
	"log"
	"unicode"
	"unicode/utf8"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Example() {
	db, err := driver.Open("file:/test.db?vfs=memdb", func(c *sqlite3.Conn) error {
		return fts5.RegisterCustom(c, func(a *fts5.Api) error {
			return a.CreateTokenizer("utf8", func(arg string) (fts5.Tokenizer, error) {
				return Utf8Tokenizer{}, nil
			})
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
	err = db.QueryRow("SELECT title FROM docs WHERE docs MATCH 'Go AND routines'").Scan(&title)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(title)
	// Output: Go Programming
}

type Utf8Tokenizer struct{}

func (Utf8Tokenizer) Tokenize(flags fts5.TokenizeFlag, text, locale string,
	xToken func(tflags fts5.TokenFlag, token string, start, end int32) error) error {

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
			err := xToken(0, "", int32(start), int32(end))
			if err != nil {
				return err
			}
			start = end
		}
	}
	return nil
}
