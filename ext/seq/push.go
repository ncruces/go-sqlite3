//go:build !coro

package seq

import "iter"

// Push takes a consumer function, and returns a yield and a stop function.
// It arranges for the consumer to be called with a Seq iterator.
// The iterator will return all the values passed to the yield function.
// The iterator will stop when the stop function is called.
func iter_Push[V any](consumer func(seq iter.Seq[V])) (
	yield func(V) bool, stop func()) {

	var (
		next = make(chan V)
		wait = make(chan struct{})
		done bool
		rcvr any
	)

	go func() {
		// recover and propagate panics
		defer func() {
			rcvr = recover()
			done = true
			wait <- struct{}{}
		}()

		wait <- struct{}{}
		consumer(func(yield func(V) bool) {
			for in := range next {
				if !yield(in) {
					break
				}
				wait <- struct{}{}
			}
		})
	}()
	<-wait

	yield = func(v V) bool {
		// yield the next value, panics if stop has been called
		next <- v
		<-wait

		// propapage panics (todo: goexit)
		if rcvr != nil {
			panic(rcvr)
		}
		return !done
	}

	stop = func() {
		// finish the iteration, panics if stop has been called
		close(next)
		<-wait

		// propapage panics (todo: goexit)
		if rcvr != nil {
			panic(rcvr)
		}
	}

	return yield, stop
}
