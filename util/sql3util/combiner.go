package sql3util

import (
	"cmp"
	"io"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// WindowCombiner is the interface an aggregate function should implement
// to be used with [WindowFromCombiner].
//
// T is the aggregate state, *T must implement the interface.
// The zero value of T must be the empty aggregate, with no rows aggregated.
type WindowCombiner[T any] interface {
	*T
	sqlite3.AggregateFunction

	// Combine merges the state of other into the receiver,
	// as if the rows aggregated by other had been added after
	// the rows aggregated by the receiver, in window order.
	//
	// Combine must be associative, but need not be commutative:
	// a.Combine(b); a.Combine(c) must be equivalent to
	// b.Combine(c); a.Combine(b).
	//
	// Both receiver and other may be empty (the zero value):
	// x.Combine(empty) must be equivalent to x.
	Combine(other T)
}

// WindowFromCombiner uses [WindowCombiner] to return an [sqlite3.AggregateConstructor]
// that implements [sqlite3.WindowFunction] using the two-stack sliding-window algorithm:
// https://orlp.net/blog/two-stack-sliding-window-aggregation/
//
// The memory efficiency of the algorithm depends on T
// efficiently representing small aggregates,
// particularly empty or single-row states.
//
// Values of T are stored and moved by value:
// T must not depend on its address (e.g. must not contain a [sync.Mutex]).
// If *T implements [io.Closer], it will be called to free resources.
//
// If maxWindow >= 0 and the window grows larger than maxWindow rows,
// the function degrades to a standard aggregate and may no longer be used as a sliding window.
// If maxWindow < 0, the window size is unbounded.
//
// It helps implement aggregate window functions over primitives
// that are combinable (or mergeable) but not invertible.
// Typical examples are mergeable sketch algorithms like HyperLogLog.
func WindowFromCombiner[T any, P WindowCombiner[T]](maxWindow int) sqlite3.AggregateConstructor {
	return func() sqlite3.AggregateFunction {
		return &twoStack[T, P]{maxWindow: maxWindow}
	}
}

type twoStack[T any, P WindowCombiner[T]] struct {
	cumAggs   []T
	values    []T
	valuesAgg [1]T
	maxWindow int
}

func (s *twoStack[T, P]) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if s.maxWindow > 0 && len(s.cumAggs)+len(s.values) >= s.maxWindow {
		// Window exceeded maxWindow; degrade to a standard aggregate.
		s.unwindow(ctx)
	}
	if s.maxWindow == 0 {
		// Degraded to a standard aggregate.
		P(&s.valuesAgg[0]).Step(ctx, arg...)
		return
	}

	s.values = append(s.values, *new(T))
	if last := len(s.values) - 1; last == 0 {
		P(&s.valuesAgg[0]).Step(ctx, arg...)
	} else {
		P(&s.values[last]).Step(ctx, arg...)
		P(&s.valuesAgg[0]).Combine(s.values[last])
	}
}

func (s *twoStack[T, P]) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if s.maxWindow == 0 {
		// Degraded to a standard aggregate.
		ctx.ResultError(errutil.ErrorString("combiner: window size exceeded maximum"))
		return
	}
	if last := len(s.cumAggs) - 1; last >= 0 {
		// Pop the oldest cumulative aggregate.
		ctx.ResultError(s.closeAll(s.cumAggs[last:]))
		clear(s.cumAggs[last:])
		s.cumAggs = s.cumAggs[:last]
		return
	}

	// Ignore the oldest value and accumulate the rest in reverse order.
	for i := len(s.values) - 2; i >= 1; i-- {
		P(&s.values[i]).Combine(s.values[i+1])
	}
	for i := len(s.values) - 1; i >= 1; i-- {
		s.cumAggs = append(s.cumAggs, s.values[i])
	}

	// Cleanup.
	ctx.ResultError(s.closeAll(s.valuesAgg[:]))
	clear(s.values)
	s.values = s.values[:0]
	s.valuesAgg = [1]T{}
}

func (s *twoStack[T, P]) Value(ctx sqlite3.Context) {
	switch {
	case len(s.cumAggs) == 0:
		P(&s.valuesAgg[0]).Value(ctx)
	case len(s.values) == 0:
		P(&s.cumAggs[len(s.cumAggs)-1]).Value(ctx)
	default:
		// Combine the oldest cumulative aggregate with the values aggregate.
		P(&s.values[0]).Combine(s.cumAggs[len(s.cumAggs)-1])
		P(&s.values[0]).Combine(s.valuesAgg[0])
		P(&s.values[0]).Value(ctx)
		ctx.ResultError(s.closeAll(s.values[:1]))
		clear(s.values[:1])
	}
}

func (s *twoStack[T, P]) Close() error {
	err := cmp.Or(
		s.closeAll(s.valuesAgg[:]),
		s.closeAll(s.values),
		s.closeAll(s.cumAggs),
	)
	*s = twoStack[T, P]{}
	return err
}

func (s *twoStack[T, P]) unwindow(ctx sqlite3.Context) {
	if last := len(s.cumAggs) - 1; last >= 0 {
		// Merge the oldest cumulative aggregate with valuesAgg.
		P(&s.cumAggs[last]).Combine(s.valuesAgg[0])
		ctx.ResultError(s.closeAll(s.valuesAgg[:]))
		s.valuesAgg[0] = s.cumAggs[last]
		s.cumAggs = s.cumAggs[:last]
	}
	// Free stacks and mark as degraded.
	ctx.ResultError(cmp.Or(
		s.closeAll(s.values),
		s.closeAll(s.cumAggs),
	))
	*s = twoStack[T, P]{valuesAgg: s.valuesAgg}
}

func (*twoStack[T, P]) closeAll(a []T) (err error) {
	if _, ok := any(P(nil)).(io.Closer); !ok {
		return nil
	}
	for i := range a {
		c := any(P(&a[i])).(io.Closer)
		err = cmp.Or(err, c.Close())
	}
	return err
}
