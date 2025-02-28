//go:build coro

package seq

import (
	"iter"
	"runtime"
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
		v          V
		done       bool
		panicValue any
		seqDone    bool // to detect Goexit
	)

	c := newcoro(func(c *coro) {
		// Recover and propagate panics from consumer.
		defer func() {
			if p := recover(); p != nil {
				panicValue = p
			} else if !seqDone {
				panicValue = goexitPanicValue
			}
			done = true
		}()

		consumer(func(yield func(V) bool) {
			for !done {
				if !yield(v) {
					break
				}
				coroswitch(c)
			}
		})
		seqDone = true
	})

	yield = func(v1 V) bool {
		if done {
			panic("yield called after stop")
		}

		v = v1
		// Yield the next value.
		coroswitch(c)

		// Propagate panics and goexits from consumer.
		if panicValue != nil {
			if panicValue == goexitPanicValue {
				// Propagate runtime.Goexit from consumer.
				runtime.Goexit()
			} else {
				panic(panicValue)
			}
		}
		return !done
	}

	stop = func() {
		if done {
			panic("stop called again")
		}

		done = true
		// Finish the iteration.
		coroswitch(c)

		// Propagate panics and goexits from consumer.
		if panicValue != nil {
			if panicValue == goexitPanicValue {
				// Propagate runtime.Goexit from consumer.
				runtime.Goexit()
			} else {
				panic(panicValue)
			}
		}
	}

	return yield, stop
}

// goexitPanicValue is a sentinel value indicating that an iterator
// exited via runtime.Goexit.
var goexitPanicValue any = new(int)
