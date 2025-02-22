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
		agg := &aggregate{
			next: make(chan []sqlite3.Value),
			wait: make(chan struct{}),
		}
		go func() {
			defer func() {
				agg.panic = recover()
				agg.done = true
				close(agg.wait)
			}()
			agg.wait <- struct{}{} // avoid any parallelism
			agg.value = processor(func(yield func([]sqlite3.Value) bool) {
				for arg := range agg.next {
					if !yield(arg) {
						break
					}
					agg.wait <- struct{}{}
				}
			})
		}()
		<-agg.wait
		return agg
	}
}

type aggregate struct {
	next  chan []sqlite3.Value
	wait  chan struct{}
	done  bool
	panic any
	value any
}

func (a *aggregate) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if !a.done {
		a.next <- arg
		<-a.wait
	}
	if a.panic != nil {
		panic(a.panic)
	}
}

func (a *aggregate) Close() error {
	if !a.done {
		close(a.next)
		<-a.wait
	}
	if a.panic != nil {
		panic(a.panic)
	}
	return nil
}

func (a *aggregate) Value(ctx sqlite3.Context) {
	a.Close() // wait for goroutine to exit

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
