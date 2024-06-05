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

const (
	median = iota
	quant_cont
	quant_disc
)

func newQuantile(kind int) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction { return &quantile{kind: kind} }
}

type quantile struct {
	nums []float64
	arg1 []byte
	kind int
}

func (q *quantile) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if a := arg[0]; a.NumericType() != sqlite3.NULL {
		q.nums = append(q.nums, a.Float())
	}
	if q.kind != median {
		q.arg1 = arg[1].Blob(q.arg1[:0])
	}
}

func (q *quantile) Value(ctx sqlite3.Context) {
	if len(q.nums) == 0 {
		return
	}

	var (
		err    error
		float  float64
		floats []float64
	)
	if q.kind == median {
		float, err = getQuantile(q.nums, 0.5, false)
		ctx.ResultFloat(float)
	} else if err = json.Unmarshal(q.arg1, &float); err == nil {
		float, err = getQuantile(q.nums, float, q.kind == quant_disc)
		ctx.ResultFloat(float)
	} else if err = json.Unmarshal(q.arg1, &floats); err == nil {
		err = getQuantiles(q.nums, floats, q.kind == quant_disc)
		ctx.ResultJSON(floats)
	}
	if err != nil {
		ctx.ResultError(fmt.Errorf("quantile: %w", err))
	}
}

func getQuantile(nums []float64, pos float64, disc bool) (float64, error) {
	if pos < 0 || pos > 1 {
		return 0, util.ErrorString("invalid pos")
	}

	i, f := math.Modf(pos * float64(len(nums)-1))
	m0 := quick.Select(nums, int(i))

	if f == 0 || disc {
		return m0, nil
	}

	m1 := slices.Min(nums[int(i)+1:])
	return math.FMA(f, m1, -math.FMA(f, m0, -m0)), nil
}

func getQuantiles(nums []float64, pos []float64, disc bool) error {
	for i := range pos {
		v, err := getQuantile(nums, pos[i], disc)
		if err != nil {
			return err
		}
		pos[i] = v
	}
	return nil
}
