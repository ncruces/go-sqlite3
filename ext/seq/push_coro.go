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

func iter_Push[V any](consumer func(seq iter.Seq[V])) (
	yield func(V) bool, stop func()) {

	var (
		next V
		done bool
		rcvr any
	)

	c := newcoro(func(c *coro) {
		// recover and propagate panics
		defer func() {
			rcvr = recover()
			done = true
		}()

		consumer(func(yield func(V) bool) {
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
			panic("yield called after stop")
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

	stop = func() {
		if done {
			panic("stop called again")
		}

		// finish the iteration
		done = true
		coroswitch(c)

		// propapage panics (goexits?)
		if rcvr != nil {
			panic(rcvr)
		}
	}

	return yield, stop
}
