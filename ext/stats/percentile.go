package stats

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/sort/quick"
)

// Compatible with:
// https://sqlite.org/src/file/ext/misc/percentile.c

const (
	median = iota
	percentile_100
	percentile_cont
	percentile_disc
)

func newPercentile(kind int) sqlite3.AggregateConstructor {
	return func() sqlite3.AggregateFunction { return &percentile{kind: kind, pct: -1} }
}

type percentile struct {
	nums []float64
	many []float64
	pcts []float64
	pct  float64
	kind int
}

func (q *percentile) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		q.nums = append(q.nums, arg[0].Float())
	}
	// Load the percentil only once.
	if q.pct >= 0 {
		return
	}
	if q.kind == median {
		q.pct = 0.5
		return
	}
	// It's either a number, or a list of numbers.
	if arg[1].NumericType() <= sqlite3.FLOAT {
		q.pct = arg[1].Float()
	} else if json.Unmarshal(arg[1].RawText(), &q.pcts) == nil {
		q.pct = 0
	}
	// Convert to quantiles.
	if q.kind == percentile_100 {
		for i := range q.pcts {
			q.pcts[i] /= 100
		}
		q.pct /= 100
	}
	// Check quantile bounds.
	if !(0 <= q.pct && q.pct <= 1) {
		ctx.ResultError(fmt.Errorf("percentile: invalid quantile: %f", q.pct))
		return
	}
	for _, pct := range q.pcts {
		if !(0 <= pct && pct <= 1) {
			ctx.ResultError(fmt.Errorf("percentile: invalid quantile: %f", pct))
			return
		}
	}
}

func (q *percentile) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		i := slices.Index(q.nums, arg[0].Float())
		l := len(q.nums) - 1
		q.nums[i] = q.nums[l]
		q.nums = q.nums[:l]
	}
}

func (q *percentile) Value(ctx sqlite3.Context) {
	if len(q.nums) == 0 {
		return
	}

	if q.pcts != nil {
		q.atMany()
		ctx.ResultJSON(q.many)
		ctx.ResultSubtype('J')
	} else {
		v, _ := q.at(q.pct, -1)
		ctx.ResultFloat(v)
	}
}

func (q *percentile) at(pct float64, prev int) (float64, int) {
	i, f := math.Modf(pct * float64(len(q.nums)-1))
	k := int(i)

	var m0 float64
	switch {
	case k < prev:
		m0 = quick.Select(q.nums[:prev], k)
	case k > prev:
		m0 = quick.Select(q.nums[prev+1:], k-prev-1)
	default:
		m0 = q.nums[k]
	}
	if f == 0 || q.kind == percentile_disc {
		return m0, k
	}
	m1 := slices.Min(q.nums[k+1:])
	return util.Lerp(m0, m1, f), k
}

func (q *percentile) atMany() {
	if len(q.many) < len(q.pcts) {
		q.many = make([]float64, len(q.pcts))
	}
	p := -1
	for i, pct := range q.pcts {
		q.many[i], p = q.at(pct, p)
	}
}
