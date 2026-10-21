package sql3util

import (
	"errors"
	"io"

	"github.com/ncruces/go-sqlite3"
)

// WindowCombiner is the interface an aggregate function should implement
// to be used with [WindowFunction].
type WindowCombiner[T sqlite3.AggregateFunction] interface {
	sqlite3.AggregateFunction

	// Combine merges the state of other into the receiver.
	// Implementations must not modify or retain other.
	Combine(other T)
}

// WindowFunction adapts a [WindowCombiner] constructor into an [sqlite3.AggregateConstructor]
// that implements [sqlite3.WindowFunction] using the two-stack sliding-window algorithm.
// If fn returns an [io.Closer], it will be called to free resources.
//
// https://orlp.net/blog/two-stack-sliding-window-aggregation/
func WindowFunction[T WindowCombiner[T]](fn func() T) sqlite3.AggregateConstructor {
	if fn == nil {
		return nil
	}
	return func() sqlite3.AggregateFunction {
		return &twoStack[T]{
			ctor:      fn,
			valuesAgg: fn(),
		}
	}
}

type twoStack[T WindowCombiner[T]] struct {
	ctor      func() T
	cumAggs   []T
	values    []T
	valuesAgg T
}

func (s *twoStack[T]) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	unit := s.ctor()
	unit.Step(ctx, arg...)
	s.values = append(s.values, unit)
	s.valuesAgg.Combine(unit)
}

func (s *twoStack[T]) Inverse(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if len(s.cumAggs) > 0 {
		last := len(s.cumAggs) - 1
		closeAgg(ctx, s.cumAggs[last])
		s.cumAggs[last] = *new(T)
		s.cumAggs = s.cumAggs[:last]
		return
	}

	closeAgg(ctx, s.values[0])
	for i := len(s.values) - 1; i >= 1; i-- {
		if i < len(s.values)-1 {
			s.values[i].Combine(s.values[i+1])
		}
		s.cumAggs = append(s.cumAggs, s.values[i])
	}
	clear(s.values)
	s.values = s.values[:0]
	closeAgg(ctx, s.valuesAgg)
	s.valuesAgg = s.ctor()
}

func (s *twoStack[T]) Value(ctx sqlite3.Context) {
	switch {
	case len(s.cumAggs) == 0:
		s.valuesAgg.Value(ctx)
	case len(s.values) == 0:
		s.cumAggs[len(s.cumAggs)-1].Value(ctx)
	default:
		tmp := s.ctor()
		tmp.Combine(s.cumAggs[len(s.cumAggs)-1])
		tmp.Combine(s.valuesAgg)
		tmp.Value(ctx)
		closeAgg(ctx, tmp)
	}
}

func (s *twoStack[T]) Close() error {
	var errs []error
	if c, ok := any(s.valuesAgg).(io.Closer); ok {
		errs = append(errs, c.Close())
	}
	for _, v := range s.values {
		if c, ok := any(v).(io.Closer); ok {
			errs = append(errs, c.Close())
		}
	}
	for _, v := range s.cumAggs {
		if c, ok := any(v).(io.Closer); ok {
			errs = append(errs, c.Close())
		}
	}
	*s = twoStack[T]{}
	return errors.Join(errs...)
}

func closeAgg[T any](ctx sqlite3.Context, a T) {
	if c, ok := any(a).(io.Closer); ok {
		if err := c.Close(); err != nil {
			ctx.ResultError(err) // notest
		}
	}
}
