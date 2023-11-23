package csv

import (
	"fmt"
	"strconv"
	"strings"
)

func getParam(arg string) (key, val string) {
	key, val, _ = strings.Cut(arg, "=")
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	return
}

func uintParam(key, val string) (int, error) {
	i, err := strconv.ParseUint(val, 10, 15)
	if err != nil {
		return 0, fmt.Errorf("csv: invalid %q parameter: %s", key, val)
	}
	return int(i), nil
}

func boolParam(key, val string) (bool, error) {
	if val == "" || val == "1" ||
		strings.EqualFold(val, "true") ||
		strings.EqualFold(val, "yes") ||
		strings.EqualFold(val, "on") {
		return true, nil
	}
	if val == "0" ||
		strings.EqualFold(val, "false") ||
		strings.EqualFold(val, "no") ||
		strings.EqualFold(val, "off") {
		return false, nil
	}
	return false, fmt.Errorf("csv: invalid %q parameter: %s", key, val)
}

func runeParam(key, val string) (rune, error) {
	r, _, tail, err := strconv.UnquoteChar(unquoteParam(val), 0)
	if tail != "" || err != nil {
		return 0, fmt.Errorf("csv: invalid %q parameter: %s", key, val)
	}
	return r, nil
}

func unquoteParam(val string) string {
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
