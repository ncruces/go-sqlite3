// Package regexp provides additional regular expression functions.
//
// It provides the following Unicode aware functions:
//   - regexp_like(text, pattern),
//   - regexp_count(text, pattern [, start]),
//   - regexp_instr(text, pattern [, start [, N [, endoption [, subexpr ]]]]),
//   - regexp_substr(text, pattern [, start [, N [, subexpr ]]]),
//   - regexp_replace(text, pattern, replacement [, start [, N ]]),
//   - and a REGEXP operator.
//
// The implementation uses Go [regexp/syntax] for regular expressions.
//
// https://github.com/nalgeon/sqlean/blob/main/docs/regexp.md
package regexp

import (
	"errors"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode/utf8"

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
		db.CreateFunction("regexp_instr", 6, flags, regexInstr),
		db.CreateFunction("regexp_substr", 2, flags, regexSubstr),
		db.CreateFunction("regexp_substr", 3, flags, regexSubstr),
		db.CreateFunction("regexp_substr", 4, flags, regexSubstr),
		db.CreateFunction("regexp_substr", 5, flags, regexSubstr),
		db.CreateFunction("regexp_replace", 3, flags, regexReplace),
		db.CreateFunction("regexp_replace", 4, flags, regexReplace),
		db.CreateFunction("regexp_replace", 5, flags, regexReplace))
}

// GlobPrefix returns a GLOB for a regular expression
// appropriate to take advantage of the [LIKE optimization]
// in a query such as:
//
//	SELECT column WHERE column GLOB :glob_prefix AND column REGEXP :regexp
//
// [LIKE optimization]: https://sqlite.org/optoverview.html#the_like_optimization
func GlobPrefix(expr string) string {
	re, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return "" // no match possible
	}
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		return "" // notest
	}

	i := &prog.Inst[prog.Start]

	var empty syntax.EmptyOp
loop1:
	for {
		switch i.Op {
		case syntax.InstFail:
			return "" // notest
		case syntax.InstCapture, syntax.InstNop:
			// skip
		case syntax.InstEmptyWidth:
			empty |= syntax.EmptyOp(i.Arg)
		default:
			break loop1
		}
		i = &prog.Inst[i.Out]
	}
	if empty&syntax.EmptyBeginText == 0 {
		return "*" // not anchored
	}

	var glob strings.Builder
loop2:
	for {
		switch i.Op {
		case syntax.InstFail:
			return "" // notest
		case syntax.InstCapture, syntax.InstEmptyWidth, syntax.InstNop:
			// skip
		case syntax.InstRune, syntax.InstRune1:
			if len(i.Rune) != 1 || syntax.Flags(i.Arg)&syntax.FoldCase != 0 {
				break loop2
			}
			switch r := i.Rune[0]; r {
			case '*', '?', '[', utf8.RuneError:
				break loop2
			default:
				glob.WriteRune(r)
			}
		default:
			break loop2
		}
		i = &prog.Inst[i.Out]
	}

	glob.WriteByte('*')
	return glob.String()
}

func load(ctx sqlite3.Context, arg []sqlite3.Value, i int) (*regexp.Regexp, error) {
	re, ok := ctx.GetAuxData(i).(*regexp.Regexp)
	if !ok {
		re, ok = arg[i].Pointer().(*regexp.Regexp)
		if !ok {
			r, err := regexp.Compile(arg[i].Text())
			if err != nil {
				return nil, err
			}
			re = r
		}
		ctx.SetAuxData(i, re)
	}
	return re, nil
}

func regex(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, arg, 0)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	text := arg[1].RawText()
	ctx.ResultBool(re.Match(text))
}

func regexLike(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, arg, 1)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	text := arg[0].RawText()
	ctx.ResultBool(re.Match(text))
}

func regexCount(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, arg, 1)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}

	text := arg[0].RawText()
	if len(arg) > 2 {
		pos := arg[2].Int()
		text = text[skip(text, pos):]
	}
	ctx.ResultInt(len(re.FindAll(text, -1)))
}

func regexSubstr(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, arg, 1)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}

	text := arg[0].RawText()
	var pos, n, subexpr int
	if len(arg) > 2 {
		pos = arg[2].Int()
	}
	if len(arg) > 3 {
		n = arg[3].Int()
	}
	if len(arg) > 4 {
		subexpr = arg[4].Int()
	}

	loc := regexFind(re, text, pos, n, subexpr)
	if loc != nil {
		ctx.ResultRawText(text[loc[0]:loc[1]])
	}
}

func regexInstr(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, arg, 1)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}

	text := arg[0].RawText()
	var pos, n, end, subexpr int
	if len(arg) > 2 {
		pos = arg[2].Int()
	}
	if len(arg) > 3 {
		n = arg[3].Int()
	}
	if len(arg) > 4 && arg[4].Bool() {
		end = 1
	}
	if len(arg) > 5 {
		subexpr = arg[5].Int()
	}

	loc := regexFind(re, text, pos, n, subexpr)
	if loc != nil {
		ctx.ResultInt(loc[end] + 1)
	}
}

func regexReplace(ctx sqlite3.Context, arg ...sqlite3.Value) {
	re, err := load(ctx, arg, 1)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}

	repl := arg[2].RawText()
	text := arg[0].RawText()
	var pos, n int
	if len(arg) > 3 {
		pos = arg[3].Int()
	}
	if len(arg) > 4 {
		n = arg[4].Int()
	}

	res := text
	pos = skip(text, pos)
	if n > 0 {
		all := re.FindAllSubmatchIndex(text[pos:], n)
		if n <= len(all) {
			loc := all[n-1]
			res = text[:pos+loc[0]]
			res = re.Expand(res, repl, text[pos:], loc)
			res = append(res, text[pos+loc[1]:]...)
		}
	} else {
		res = append(text[:pos], re.ReplaceAll(text[pos:], repl)...)
	}
	ctx.ResultRawText(res)
}

func regexFind(re *regexp.Regexp, text []byte, pos, n, subexpr int) (loc []int) {
	pos = skip(text, pos)
	text = text[pos:]

	if n <= 1 {
		if subexpr == 0 {
			loc = re.FindIndex(text)
		} else {
			loc = re.FindSubmatchIndex(text)
		}
	} else {
		if subexpr == 0 {
			all := re.FindAllIndex(text, n)
			if n <= len(all) {
				loc = all[n-1]
			}
		} else {
			all := re.FindAllSubmatchIndex(text, n)
			if n <= len(all) {
				loc = all[n-1]
			}
		}
	}

	if 2+2*subexpr <= len(loc) {
		loc = loc[2*subexpr : 2+2*subexpr]
		loc[0] += pos
		loc[1] += pos
		return loc
	}
	return nil
}

func skip(text []byte, start int) int {
	for pos := range string(text) {
		if start--; start <= 0 {
			return pos
		}
	}
	return len(text)
}
