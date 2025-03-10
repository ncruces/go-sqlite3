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
//   - the REGEXP operator uses Go [regexp/syntax];
//   - collation sequences use [collate].
//
// It also provides (approximately) from PostgreSQL:
//   - casefold(),
//   - initcap(),
//   - normalize(),
//   - unaccent().
//
// Expect subtle differences (e.g.) in the handling of Turkish case folding.
//
// [ICU extension]: https://sqlite.org/src/dir/ext/icu
package unicode

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Set RegisterLike to false to not register a Unicode aware LIKE operator.
// Overriding the built-in LIKE operator disables the [LIKE optimization].
//
// [LIKE optimization]: https://sqlite.org/optoverview.html#the_like_optimization
var RegisterLike = true

// Register registers Unicode aware functions for a database connection.
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	var lkfn sqlite3.ScalarFunction
	if RegisterLike {
		lkfn = like
	}
	return errors.Join(
		db.CreateFunction("like", 2, flags, lkfn),
		db.CreateFunction("like", 3, flags, lkfn),
		db.CreateFunction("upper", 1, flags, upper),
		db.CreateFunction("upper", 2, flags, upper),
		db.CreateFunction("lower", 1, flags, lower),
		db.CreateFunction("lower", 2, flags, lower),
		db.CreateFunction("regexp", 2, flags, regex),
		db.CreateFunction("initcap", 1, flags, initcap),
		db.CreateFunction("initcap", 2, flags, initcap),
		db.CreateFunction("casefold", 1, flags, casefold),
		db.CreateFunction("unaccent", 1, flags, unaccent),
		db.CreateFunction("normalize", 1, flags, normalize),
		db.CreateFunction("normalize", 2, flags, normalize),
		db.CreateFunction("icu_load_collation", 2, sqlite3.DIRECTONLY,
			func(ctx sqlite3.Context, arg ...sqlite3.Value) {
				name := arg[1].Text()
				if name == "" {
					return
				}

				err := RegisterCollation(ctx.Conn(), arg[0].Text(), name)
				if err != nil {
					ctx.ResultError(err)
					return // notest
				}
			}))
}

// RegisterCollation registers a Unicode collation sequence for a database connection.
func RegisterCollation(db *sqlite3.Conn, locale, name string) error {
	tag, err := language.Parse(locale)
	if err != nil {
		return err
	}
	return db.CreateCollation(name, collate.New(tag).Compare)
}

// RegisterCollationsNeeded registers Unicode collation sequences on demand for a database connection.
func RegisterCollationsNeeded(db *sqlite3.Conn) error {
	return db.CollationNeeded(func(db *sqlite3.Conn, name string) {
		if tag, err := language.Parse(name); err == nil {
			db.CreateCollation(name, collate.New(tag).Compare)
		}
	})
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
			return // notest
		}
		cs = cases.Upper(t)
		ctx.SetAuxData(1, cs)
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
			return // notest
		}
		cs = cases.Lower(t)
		ctx.SetAuxData(1, cs)
	}
	ctx.ResultRawText(cs.Bytes(arg[0].RawText()))
}

func initcap(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if len(arg) == 1 {
		ctx.ResultRawText(bytes.Title(arg[0].RawText()))
		return
	}
	cs, ok := ctx.GetAuxData(1).(cases.Caser)
	if !ok {
		t, err := language.Parse(arg[1].Text())
		if err != nil {
			ctx.ResultError(err)
			return // notest
		}
		cs = cases.Title(t)
		ctx.SetAuxData(1, cs)
	}
	ctx.ResultRawText(cs.Bytes(arg[0].RawText()))
}

func casefold(ctx sqlite3.Context, arg ...sqlite3.Value) {
	ctx.ResultRawText(cases.Fold().Bytes(arg[0].RawText()))
}

func unaccent(ctx sqlite3.Context, arg ...sqlite3.Value) {
	unaccent := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, err := transform.Bytes(unaccent, arg[0].RawText())
	if err != nil {
		ctx.ResultError(err) // notest
	} else {
		ctx.ResultRawText(res)
	}
}

func normalize(ctx sqlite3.Context, arg ...sqlite3.Value) {
	form := norm.NFC
	if len(arg) > 1 {
		switch strings.ToUpper(arg[1].Text()) {
		case "NFC":
			//
		case "NFD":
			form = norm.NFD
		case "NFKC":
			form = norm.NFKC
		case "NFKD":
			form = norm.NFKD
		default:
			ctx.ResultError(util.ErrorString("unicode: invalid form"))
			return
		}
	}
	res, _, err := transform.Bytes(form, arg[0].RawText())
	if err != nil {
		ctx.ResultError(err) // notest
	} else {
		ctx.ResultRawText(res)
	}
}

func regex(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, ok := ctx.GetAuxData(0).(*regexp.Regexp)
	if !ok {
		re, ok = arg[0].Pointer().(*regexp.Regexp)
		if !ok {
			r, err := regexp.Compile(arg[0].Text())
			if err != nil {
				ctx.ResultError(err)
				return // notest
			}
			re = r
		}
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
	_ = arg[1] // bounds check

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
