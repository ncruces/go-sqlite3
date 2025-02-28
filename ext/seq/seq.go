package seq

import (
	"fmt"
	"iter"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func Aggregate(processor func(iter.Seq[[]sqlite3.Value]) any) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction {
		a := &aggregate{}
		a.yield, a.stop = iter_Push(func(seq iter.Seq[[]sqlite3.Value]) {
			a.value = processor(seq)
		})
		return a
	}
}

type aggregate struct {
	yield func([]sqlite3.Value) bool
	stop  func()
	done  bool
	value any
}

func (a *aggregate) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if !a.done {
		a.done = !a.yield(arg)
	}
}

func (a *aggregate) Close() error {
	if !a.done {
		a.stop()
		a.done = true
	}
	return nil
}

func (a *aggregate) Value(ctx sqlite3.Context) {
	if !a.done {
		a.stop()
		a.done = true
	}
	switch res := a.value.(type) {
	case bool:
		ctx.ResultBool(res)
	case int:
		ctx.ResultInt(res)
	case int64:
		ctx.ResultInt64(res)
	case float64:
		ctx.ResultFloat(res)
	case string:
		ctx.ResultText(res)
	case []byte:
		ctx.ResultBlob(res)
	case time.Time:
		ctx.ResultTime(res, sqlite3.TimeFormatDefault)
	case sqlite3.ZeroBlob:
		ctx.ResultZeroBlob(int64(res))
	case sqlite3.Value:
		ctx.ResultValue(res)
	case util.JSON:
		ctx.ResultJSON(res.Value)
	case util.PointerUnwrap:
		ctx.ResultPointer(util.UnwrapPointer(res))
	case error:
		ctx.ResultError(res)
	case nil:
		ctx.ResultNull()
	default:
		ctx.ResultError(fmt.Errorf("aggregate returned:%.0w %T",
			sqlite3.MISMATCH, res))
	}
}
