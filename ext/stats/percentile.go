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

func newPercentile(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &percentile{kind: kind} }
}

type percentile struct {
	nums []float64
	arg1 []byte
	kind int
}

func (q *percentile) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.NumericType() != sqlite3.NULL {
		q.nums = append(q.nums, f)
	}
	if q.kind != median && q.arg1 == nil {
		q.arg1 = append(q.arg1, arg[1].RawText()...)
	}
}

func (q *percentile) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	a := arg[0]
	f := a.Float()
	if f != 0.0 || a.NumericType() != sqlite3.NULL {
		i := slices.Index(q.nums, f)
		l := len(q.nums) - 1
		q.nums[i] = q.nums[l]
		q.nums = q.nums[:l]
	}
}

func (q *percentile) Value(ctx sqlite3.Context) {
	if len(q.nums) == 0 {
		return
	}

	var (
		err    error
		float  float64
		floats []float64
	)
	if q.kind == median {
		float, err = q.at(0.5)
		ctx.ResultFloat(float)
	} else if err = json.Unmarshal(q.arg1, &float); err == nil {
		float, err = q.at(float)
		ctx.ResultFloat(float)
	} else if err = json.Unmarshal(q.arg1, &floats); err == nil {
		err = q.atMore(floats)
		ctx.ResultJSON(floats)
	}
	if err != nil {
		ctx.ResultError(fmt.Errorf("percentile: %w", err)) // notest
	}
}

func (q *percentile) at(pos float64) (float64, error) {
	if q.kind == percentile_100 {
		pos = pos / 100
	}
	if pos < 0 || pos > 1 {
		return 0, util.ErrorString("invalid pos")
	}

	i, f := math.Modf(pos * float64(len(q.nums)-1))
	m0 := quick.Select(q.nums, int(i))

	if f == 0 || q.kind == percentile_disc {
		return m0, nil
	}

	m1 := slices.Min(q.nums[int(i)+1:])
	return util.Lerp(m0, m1, f), nil
}

func (q *percentile) atMore(pos []float64) error {
	for i := range pos {
		v, err := q.at(pos[i])
		if err != nil {
			return err
		}
		pos[i] = v
	}
	return nil
}
