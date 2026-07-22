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
	if q.pct < 0 || q.pct > 1 {
		ctx.ResultError(fmt.Errorf("percentile: invalid quantile: %f", q.pct))
		return
	}
	for _, pct := range q.pcts {
		if pct < 0 || pct > 1 {
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
	} else {
		ctx.ResultFloat(q.at(q.pct))
	}
}

func (q *percentile) at(pct float64) float64 {
	i, f := math.Modf(pct * float64(len(q.nums)-1))
	m0 := quick.Select(q.nums, int(i))

	if f == 0 || q.kind == percentile_disc {
		return m0
	}

	m1 := slices.Min(q.nums[int(i)+1:])
	return util.Lerp(m0, m1, f)
}

func (q *percentile) atMany() {
	if len(q.many) < len(q.pcts) {
		q.many = make([]float64, len(q.pcts))
	}
	for i, pct := range q.pcts {
		q.many[i] = q.at(pct)
	}
}
