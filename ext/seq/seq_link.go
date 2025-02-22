//go:build linkname

package seq

import (
	"fmt"
	"iter"
	"time"
	_ "unsafe"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

type coro struct{}

//go:linkname newcoro runtime.newcoro
func newcoro(func(*coro)) *coro

//go:linkname coroswitch runtime.coroswitch
func coroswitch(*coro)

func Aggregate(processor func(iter.Seq[[]sqlite3.Value]) any) func() sqlite3.AggregateFunction {
	return func() sqlite3.AggregateFunction {
		agg := &aggregate{}
		agg.coro = newcoro(func(c *coro) {
			defer func() {
				agg.panic = recover()
				agg.done = true
			}()
			agg.value = processor(func(yield func([]sqlite3.Value) bool) {
				for !agg.done {
					if !yield(agg.next) {
						break
					}
					coroswitch(c)
				}
			})
		})
		return agg
	}
}

type aggregate struct {
	*coro
	next  []sqlite3.Value
	done  bool
	panic any
	value any
}

func (a *aggregate) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if !a.done {
		a.next = arg
		coroswitch(a.coro)
	}
	if a.panic != nil {
		panic(a.panic)
	}
}

func (a *aggregate) Close() error {
	if !a.done {
		a.done = true
		coroswitch(a.coro)
	}
	if a.panic != nil {
		panic(a.panic)
	}
	return nil
}

func (a *aggregate) Value(ctx sqlite3.Context) {
	a.Close() // wait for coroutine to exit

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
