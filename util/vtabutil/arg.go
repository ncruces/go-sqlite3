package vtabutil

import "strings"

// NamedArg splits an named arg into a key and value,
// around an equals sign.
// Spaces are trimmed around both key and value.
func NamedArg(arg string) (key, val string) {
	key, val, _ = strings.Cut(arg, "=")
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	return
}

// Unquote unquotes a string.
func Unquote(val string) string {
	if len(val) < 2 {
		return val
	}
	if val[0] != val[len(val)-1] {
		return val
	}
	var old, new string
	switch val[0] {
	default:
		return val
	case '"':
		old, new = `""`, `"`
	case '\'':
		old, new = `''`, `'`
	}
	return strings.ReplaceAll(val[1:len(val)-1], old, new)
}
