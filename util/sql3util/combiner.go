package sql3util

import (
	"cmp"
	"io"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// WindowCombiner is the interface an aggregate function should implement
// to be used with [WindowFromCombiner].
type WindowCombiner[T sqlite3.AggregateFunction] interface {
	sqlite3.AggregateFunction

	// Combine merges the state of other into the receiver,
	// as if the rows aggregated by other had been added after
	// the rows aggregated by the receiver, in window order.
	//
	// Combine must be associative, but need not be commutative:
	// a.Combine(b); a.Combine(c) must be equivalent to
	// b.Combine(c); a.Combine(b).
	//
	// Both receiver and other may be empty, with no rows aggregated:
	// x.Combine(empty) must be equivalent to x.
	//
	// Implementations must not modify or retain other.
	Combine(other T)
}

// WindowFromCombiner adapts a [WindowCombiner] constructor into an [sqlite3.AggregateConstructor]
// that implements [sqlite3.WindowFunction] using the two-stack sliding-window algorithm:
// https://orlp.net/blog/two-stack-sliding-window-aggregation/
//
// The memory efficiency of the algorithm depends on [WindowCombiner]
// efficiently representing small aggregates,
// particularly empty or single-row states.
//
// If maxWindow > 0 and the window grows larger than maxWindow rows,
// the function degrades to a standard aggregate and may no longer be used as a sliding window.
// If T implements [io.Closer], it will be called to free resources.
//
// It helps implement aggregate window functions over primitives
// that are combinable (or mergeable) but not invertible.
// Typical examples are mergeable sketch algorithms like HyperLogLog.
func WindowFromCombiner[T WindowCombiner[T]](maxWindow int, fn func() T) sqlite3.AggregateConstructor {
	if fn == nil {
		return nil
	}
	return func() sqlite3.AggregateFunction {
		return &twoStack[T]{
			ctor:      fn,
			valuesAgg: fn(),
			maxWindow: maxWindow,
		}
	}
}

type twoStack[T WindowCombiner[T]] struct {
	ctor      func() T
	cumAggs   []T
	values    []T
	valuesAgg T
	maxWindow int
}

func (s *twoStack[T]) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if s.maxWindow > 0 && len(s.cumAggs)+len(s.values) >= s.maxWindow {
		// Window exceeded maxWindow; degrade to a standard aggregate.
		s.unwindow(ctx)
	}
	if s.ctor == nil {
		// Degraded to a standard aggregate.
		s.valuesAgg.Step(ctx, arg...)
		return
	}

	unit := s.ctor()
	s.values = append(s.values, unit)
	unit.Step(ctx, arg...)
	s.valuesAgg.Combine(unit)
}

func (s *twoStack[T]) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if s.ctor == nil {
		// Degraded to a standard aggregate.
		ctx.ResultError(errutil.ErrorString("combiner: window size exceeded maximum"))
		return
	}
	if len(s.cumAggs) > 0 {
		// Pop the oldest cumulative aggregate.
		last := len(s.cumAggs) - 1
		closeAgg(ctx, s.cumAggs[last])
		clear(s.cumAggs[last:])
		s.cumAggs = s.cumAggs[:last]
		return
	}

	// Ignore the oldest value and accumulate the rest in reverse order.
	for i := len(s.values) - 2; i >= 1; i-- {
		s.values[i].Combine(s.values[i+1])
	}
	for i := len(s.values) - 1; i >= 1; i-- {
		s.cumAggs = append(s.cumAggs, s.values[i])
	}

	// Cleanup.
	closeAgg(ctx, s.valuesAgg, s.values[0])
	clear(s.values)
	s.values = s.values[:0]
	s.valuesAgg = s.ctor()
}

func (s *twoStack[T]) Value(ctx sqlite3.Context) {
	switch {
	case len(s.cumAggs) == 0:
		s.valuesAgg.Value(ctx)
	case len(s.values) == 0:
		s.cumAggs[len(s.cumAggs)-1].Value(ctx)
	default:
		// Combine the oldest cumulative aggregate with the values aggregate.
		tmp := s.ctor()
		defer closeAgg(ctx, tmp)
		tmp.Combine(s.cumAggs[len(s.cumAggs)-1])
		tmp.Combine(s.valuesAgg)
		tmp.Value(ctx)
	}
}

func (s *twoStack[T]) Close() error {
	var err error
	if c, ok := any(s.valuesAgg).(io.Closer); ok {
		err = cmp.Or(err, c.Close())
	}
	for _, v := range s.values {
		if c, ok := any(v).(io.Closer); ok {
			err = cmp.Or(err, c.Close())
		}
	}
	for _, v := range s.cumAggs {
		if c, ok := any(v).(io.Closer); ok {
			err = cmp.Or(err, c.Close())
		}
	}
	*s = twoStack[T]{}
	return err
}

func (s *twoStack[T]) unwindow(ctx sqlite3.Context) {
	if last := len(s.cumAggs) - 1; last >= 0 {
		// Merge the oldest cumulative aggregate with valuesAgg.
		agg := s.cumAggs[last]
		agg.Combine(s.valuesAgg)
		closeAgg(ctx, s.valuesAgg)
		s.valuesAgg = agg
		s.cumAggs = s.cumAggs[:last]
	}
	// Free stacks and clear ctor to mark as degraded.
	closeAgg(ctx, s.values...)
	closeAgg(ctx, s.cumAggs...)
	*s = twoStack[T]{valuesAgg: s.valuesAgg}
}

func closeAgg[T any](ctx sqlite3.Context, a ...T) {
	var err error
	for _, v := range a {
		if c, ok := any(v).(io.Closer); ok {
			if e := c.Close(); err == nil {
				err = e
			}
		}
	}
	ctx.ResultError(err)
}
