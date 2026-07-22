package stats

import (
	"math"

	"github.com/ncruces/go-sqlite3"
)

func cot(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		ctx.ResultFloat(1 / math.Tan(arg[0].Float()))
	}
}

func cbrt(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		ctx.ResultFloat(math.Cbrt(arg[0].Float()))
	}
}
