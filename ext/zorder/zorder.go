// Package zorder provides functions for z-order transformations.
//
// https://sqlite.org/src/doc/tip/ext/misc/zorder.c
package zorder

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Register registers the zorder and unzorder SQL functions.
func Register(db *sqlite3.Conn) error {
	flags := sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateFunction("zorder", -1, flags, zorder),
		db.CreateFunction("unzorder", 3, flags, unzorder))
}

func zorder(ctx sqlite3.Context, arg ...sqlite3.Value) {
	var x [63]int64
	if len(arg) > len(x) {
		ctx.ResultError(util.ErrorString("zorder: too many parameters"))
		return
	}
	for i := range arg {
		x[i] = arg[i].Int64()
	}

	var z int64
	if len(arg) > 0 {
		for i := range x {
			j := i % len(arg)
			z |= (x[j] & 1) << i
			x[j] >>= 1
		}
	}

	for i := range arg {
		if x[i] != 0 {
			ctx.ResultError(util.ErrorString("zorder: parameter too large"))
			return
		}
	}
	ctx.ResultInt64(z)
}

func unzorder(ctx sqlite3.Context, arg ...sqlite3.Value) {
	i := arg[2].Int64()
	n := arg[1].Int64()
	z := arg[0].Int64()

	var k int
	var x int64
	for j := i; j < 63; j += n {
		x |= ((z >> j) & 1) << k
		k++
	}
	ctx.ResultInt64(x)
}
