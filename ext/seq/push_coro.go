//go:build coro

package seq

import (
	"iter"
	_ "unsafe"
)

type coro struct{}

//go:linkname newcoro runtime.newcoro
func newcoro(func(*coro)) *coro

//go:linkname coroswitch runtime.coroswitch
func coroswitch(*coro)

func iter_Push[V, R any](processor func(seq iter.Seq[V]) R) (
	yield func(V) bool, stop func() R) {

	var (
		next V
		done bool
		rcvr any
		rslt R
	)

	c := newcoro(func(c *coro) {
		// recover and propagate panics
		defer func() {
			rcvr = recover()
			done = true
		}()
		rslt = processor(func(yield func(V) bool) {
			for !done {
				if !yield(next) {
					break
				}
				coroswitch(c)
			}
		})
	})

	yield = func(v V) bool {
		if done {
			// maybe panic instead?
			return true
		}

		// yield the next value
		next = v
		coroswitch(c)

		// propapage panics (goexits?)
		if rcvr != nil {
			panic(rcvr)
		}
		return !done
	}

	stop = func() R {
		if done {
			// maybe panic instead?
			return rslt
		}

		// finish the iteration
		done = true
		coroswitch(c)

		// propapage panics (goexits?)
		if rcvr != nil {
			panic(rcvr)
		}
		return rslt
	}

	return yield, stop
}
