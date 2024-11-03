// Package regexp provides additional regular expression functions.
//
// It provides the following Unicode aware functions:
//   - regexp_like(),
//   - regexp_count(),
//   - regexp_instr(),
//   - regexp_substr(),
//   - regexp_replace(),
//   - and a REGEXP operator.
//
// The implementation uses Go [regexp/syntax] for regular expressions.
//
// https://github.com/nalgeon/sqlean/blob/main/docs/regexp.md
package regexp

import (
	"errors"
	"regexp"

	"github.com/ncruces/go-sqlite3"
)

// Register registers Unicode aware functions for a database connection.
func Register(db *sqlite3.Conn) error {
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateFunction("regexp", 2, flags, regex),
		db.CreateFunction("regexp_like", 2, flags, regexLike),
		db.CreateFunction("regexp_count", 2, flags, regexCount),
		db.CreateFunction("regexp_count", 3, flags, regexCount),
		db.CreateFunction("regexp_instr", 2, flags, regexInstr),
		db.CreateFunction("regexp_instr", 3, flags, regexInstr),
		db.CreateFunction("regexp_instr", 4, flags, regexInstr),
		db.CreateFunction("regexp_instr", 5, flags, regexInstr),
		db.CreateFunction("regexp_substr", 2, flags, regexSubstr),
		db.CreateFunction("regexp_substr", 3, flags, regexSubstr),
		db.CreateFunction("regexp_substr", 4, flags, regexSubstr),
		db.CreateFunction("regexp_replace", 3, flags, regexReplace),
		db.CreateFunction("regexp_replace", 4, flags, regexReplace))
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
		return // notest
	}
	text := arg[1].RawText()
	ctx.ResultBool(re.Match(text))
}

func regexLike(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	text := arg[0].RawText()
	ctx.ResultBool(re.Match(text))
}

func regexCount(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	text := arg[0].RawText()
	if len(arg) > 2 {
		pos := arg[2].Int()
		_, text = split(text, pos)
	}
	ctx.ResultInt(len(re.FindAll(text, -1)))
}

func regexSubstr(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	text := arg[0].RawText()
	if len(arg) > 2 {
		pos := arg[2].Int()
		_, text = split(text, pos)
	}
	n := 0
	if len(arg) > 3 {
		n = arg[3].Int()
	}

	var res []byte
	if n <= 1 {
		res = re.Find(text)
	} else {
		all := re.FindAll(text, n)
		if n <= len(all) {
			res = all[n-1]
		}
	}
	ctx.ResultRawText(res)
}

func regexInstr(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	pos := 1
	text := arg[0].RawText()
	if len(arg) > 2 {
		pos = arg[2].Int()
		_, text = split(text, pos)
	}
	n := 0
	if len(arg) > 3 {
		n = arg[3].Int()
	}

	var loc []int
	if n <= 1 {
		loc = re.FindIndex(text)
	} else {
		all := re.FindAllIndex(text, n)
		if n <= len(all) {
			loc = all[n-1]
		}
	}
	if loc == nil {
		return
	}

	end := 0
	if len(arg) > 4 && arg[4].Bool() {
		end = 1
	}
	ctx.ResultInt(pos + loc[end])
}

func regexReplace(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, 1, arg[1].Text())
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	var head, tail []byte
	tail = arg[0].RawText()
	if len(arg) > 3 {
		pos := arg[3].Int()
		head, tail = split(tail, pos)
	}
	tail = re.ReplaceAll(tail, arg[2].RawText())
	if head != nil {
		tail = append(head, tail...)
	}
	ctx.ResultRawText(tail)
}

func split(s []byte, i int) (head, tail []byte) {
	for pos := range string(s) {
		if i--; i <= 0 {
			return s[:pos:pos], s[pos:]
		}
	}
	return s, nil
}
