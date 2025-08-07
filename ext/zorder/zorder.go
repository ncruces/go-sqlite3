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
	const flags = sqlite3.DETERMINISTIC | sqlite3.INNOCUOUS
	return errors.Join(
		db.CreateFunction("zorder", -1, flags, zorder),
		db.CreateFunction("unzorder", 3, flags, unzorder))
}

func zorder(ctx sqlite3.Context, arg ...sqlite3.Value) {
	var x [24]int64
	if n := len(arg); n < 2 || n > 24 {
		ctx.ResultError(util.ErrorString("zorder: needs between 2 and 24 dimensions"))
		return
	}
	for i := range arg {
		x[i] = arg[i].Int64()
	}

	var z int64
	for i := range 63 {
		j := i % len(arg)
		z |= (x[j] & 1) << i
		x[j] >>= 1
	}

	for i := range arg {
		if x[i] != 0 {
			ctx.ResultError(util.ErrorString("zorder: argument out of range"))
			return
		}
	}
	ctx.ResultInt64(z)
}

func unzorder(ctx sqlite3.Context, arg ...sqlite3.Value) {
	i := arg[2].Int64()
	n := arg[1].Int64()
	z := arg[0].Int64()

	if n < 2 || n > 24 {
		ctx.ResultError(util.ErrorString("unzorder: needs between 2 and 24 dimensions"))
		return
	}
	if i < 0 || i >= n {
		ctx.ResultError(util.ErrorString("unzorder: index out of range"))
		return
	}
	if z < 0 {
		ctx.ResultError(util.ErrorString("unzorder: argument out of range"))
		return
	}

	var k int
	var x int64
	for j := i; j < 63; j += n {
		x |= ((z >> j) & 1) << k
		k++
	}
	ctx.ResultInt64(x)
}
