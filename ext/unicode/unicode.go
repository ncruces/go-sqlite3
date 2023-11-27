// Package unicode provides an alternative to the SQLite ICU extension.
//
// Like the [ICU extension], it provides Unicode aware:
//   - upper() and lower() functions,
//   - LIKE and REGEXP operators,
//   - collation sequences.
//
// The implementation is not 100% compatible with the [ICU extension]:
//   - upper() and lower() use [strings.ToUpper], [strings.ToLower] and [cases];
//   - the LIKE operator follows [strings.EqualFold] rules;
//   - the REGEXP operator uses Go [regex/syntax];
//   - collation sequences use [collate].
//
// Expect subtle differences (e.g.) in the handling of Turkish case folding.
//
// [ICU extension]: https://sqlite.org/src/dir/ext/icu
package unicode

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"golang.org/x/text/cases"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Register registers Unicode aware functions for a database connection.
func Register(db *sqlite3.Conn) {
	flags := sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS

	db.CreateFunction("like", 2, flags, like)
	db.CreateFunction("like", 3, flags, like)
	db.CreateFunction("upper", 1, flags, upper)
	db.CreateFunction("upper", 2, flags, upper)
	db.CreateFunction("lower", 1, flags, lower)
	db.CreateFunction("lower", 2, flags, lower)
	db.CreateFunction("regexp", 2, flags, regex)
	db.CreateFunction("icu_load_collation", 2, sqlite3.DIRECTONLY,
		func(ctx sqlite3.Context, arg ...sqlite3.Value) {
			name := arg[1].Text()
			if name == "" {
				return
			}

			err := RegisterCollation(db, arg[0].Text(), name)
			if err != nil {
				ctx.ResultError(err)
				return
			}
		})
}

// RegisterCollation registers a Unicode collation sequence for a database connection.
func RegisterCollation(db *sqlite3.Conn, locale, name string) error {
	tag, err := language.Parse(locale)
	if err != nil {
		return err
	}
	return db.CreateCollation(name, collate.New(tag).Compare)
}

func upper(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if len(arg) == 1 {
		ctx.ResultRawText(bytes.ToUpper(arg[0].RawText()))
		return
	}
	cs, ok := ctx.GetAuxData(1).(cases.Caser)
	if !ok {
		t, err := language.Parse(arg[1].Text())
		if err != nil {
			ctx.ResultError(err)
			return
		}
		c := cases.Upper(t)
		ctx.SetAuxData(1, c)
		cs = c
	}
	ctx.ResultRawText(cs.Bytes(arg[0].RawText()))
}

func lower(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if len(arg) == 1 {
		ctx.ResultRawText(bytes.ToLower(arg[0].RawText()))
		return
	}
	cs, ok := ctx.GetAuxData(1).(cases.Caser)
	if !ok {
		t, err := language.Parse(arg[1].Text())
		if err != nil {
			ctx.ResultError(err)
			return
		}
		c := cases.Lower(t)
		ctx.SetAuxData(1, c)
		cs = c
	}
	ctx.ResultRawText(cs.Bytes(arg[0].RawText()))
}

func regex(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, ok := ctx.GetAuxData(0).(*regexp.Regexp)
	if !ok {
		r, err := regexp.Compile(arg[0].Text())
		if err != nil {
			ctx.ResultError(err)
			return
		}
		re = r
		ctx.SetAuxData(0, re)
	}
	ctx.ResultBool(re.Match(arg[1].RawText()))
}

func like(ctx sqlite3.Context, arg ...sqlite3.Value) {
	escape := rune(-1)
	if len(arg) == 3 {
		var size int
		b := arg[2].RawText()
		escape, size = utf8.DecodeRune(b)
		if size != len(b) {
			ctx.ResultError(util.ErrorString("ESCAPE expression must be a single character"))
			return
		}
	}

	type likeData struct {
		*regexp.Regexp
		escape rune
	}

	re, ok := ctx.GetAuxData(0).(likeData)
	if !ok || re.escape != escape {
		re = likeData{
			regexp.MustCompile(like2regex(arg[0].Text(), escape)),
			escape,
		}
		ctx.SetAuxData(0, re)
	}
	ctx.ResultBool(re.Match(arg[1].RawText()))
}

func like2regex(pattern string, escape rune) string {
	var re strings.Builder
	start := 0
	literal := false
	re.Grow(len(pattern) + 10)
	re.WriteString(`(?is)\A`) // case insensitive, . matches any character
	for i, r := range pattern {
		if start < 0 {
			start = i
		}
		if literal {
			literal = false
			continue
		}
		var symbol string
		switch r {
		case '_':
			symbol = `.`
		case '%':
			symbol = `.*`
		case escape:
			literal = true
		default:
			continue
		}
		re.WriteString(regexp.QuoteMeta(pattern[start:i]))
		re.WriteString(symbol)
		start = -1
	}
	if start >= 0 {
		re.WriteString(regexp.QuoteMeta(pattern[start:]))
	}
	re.WriteString(`\z`)
	return re.String()
}
