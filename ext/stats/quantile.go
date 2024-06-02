package stats

import (
	"math"
	"slices"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/sort/quick"
)

const (
	median = iota
	quant_cont
	quant_disc
)

func newQuantile(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &quantile{kind: kind} }
}

type quantile struct {
	kind int
	pos  float64
	list []float64
}

func (q *quantile) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.NumericType() != sqlite3.NULL {
		q.list = append(q.list, a.Float())
	}
	if q.kind != median {
		q.pos = arg[1].Float()
	}
}

func (q *quantile) Value(ctx sqlite3.Context) {
	if len(q.list) == 0 {
		return
	}
	if q.kind == median {
		q.pos = 0.5
	}
	if q.pos < 0 || q.pos > 1 {
		ctx.ResultError(util.ErrorString("quantile: invalid pos"))
		return
	}

	i, f := math.Modf(q.pos * float64(len(q.list)-1))
	m0 := quick.Select(q.list, int(i))

	if f == 0 || q.kind == quant_disc {
		ctx.ResultFloat(m0)
		return
	}

	m1 := slices.Min(q.list[int(i)+1:])
	ctx.ResultFloat(math.FMA(f, m1, -math.FMA(f, m0, -m0)))
}
