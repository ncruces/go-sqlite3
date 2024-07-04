// Package regexp provides additional regular expression functions.
//
// It provides the following Unicode aware functions:
//   - regexp_like(),
//   - regexp_substr(),
//   - regexp_replace(),
//   - and a REGEXP operator.
//
// The implementation uses Go [regexp/syntax] for regular expressions.
//
// https://github.com/nalgeon/sqlean/blob/main/docs/regexp.md
package regexp

import (
	"regexp"

	"github.com/ncruces/go-sqlite3"
)

// Register registers Unicode aware functions for a database connection.
func Register(db *sqlite3.Conn) {
	flags := sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS

	db.CreateFunction("regexp", 2, flags, regex)
	db.CreateFunction("regexp_like", 2, flags, regexLike)
	db.CreateFunction("regexp_substr", 2, flags, regexSubstr)
	db.CreateFunction("regexp_replace", 3, flags, regexReplace)
}

func load(ctx sqlite3.Context, i int, expr string) (*regexp.Regexp, error) {
	re, ok := ctx.GetAuxData(i).(*regexp.Regexp)
	if !ok {
		r, err := regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
		re = r
		ctx.SetAuxData(0, r)
	}
	return re, nil
}

func regex(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 0, arg[0].Text())
	if err != nil {
		ctx.ResultError(err)
	} else {
		ctx.ResultBool(re.Match(arg[1].RawText()))
	}
}

func regexLike(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
	} else {
		ctx.ResultBool(re.Match(arg[0].RawText()))
	}
}

func regexSubstr(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
	} else {
		ctx.ResultRawText(re.Find(arg[0].RawText()))
	}
}

func regexReplace(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
	} else {
		ctx.ResultRawText(re.ReplaceAll(arg[0].RawText(), arg[2].RawText()))
	}
}
