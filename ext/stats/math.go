package stats

import (
	"math"

	"github.com/ncruces/go-sqlite3"
)

func cot(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if f := arg[0].Float(); f != 0.0 {
		ctx.ResultFloat(1 / math.Tan(f))
	}
}

func cbrt(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.Type() != sqlite3.NULL {
		ctx.ResultFloat(math.Cbrt(f))
	}
}
