package csv

import (
	"fmt"
	"strconv"

	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func uintArg(key, val string) (int, error) {
	i, err := strconv.ParseUint(val, 10, 15)
	if err != nil {
		return 0, fmt.Errorf("csv: invalid %q parameter: %s", key, val)
	}
	return int(i), nil
}

func boolArg(key, val string) (bool, error) {
	if val == "" {
		return true, nil
	}
	b, ok := sql3util.ParseBool(val)
	if ok {
		return b, nil
	}
	return false, fmt.Errorf("csv: invalid %q parameter: %s", key, val)
}

func runeArg(key, val string) (rune, error) {
	r, _, tail, err := strconv.UnquoteChar(sql3util.Unquote(val), 0)
	if tail != "" || err != nil {
		return 0, fmt.Errorf("csv: invalid %q parameter: %s", key, val)
	}
	return r, nil
}
