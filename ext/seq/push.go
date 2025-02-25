//go:build !coro

package seq

import "iter"

// Push takes a processor, a function that operates on a Seq and returns a result,
// and returns two functions:
//   - yield, a function that pushes values to seq
//   - stop, a function that stops iteration and collects the result
//
// These functions can be used to implement arbitrary “push-style” iteration interfaces
// with processors designed to operate on a Seq.
func iter_Push[V, R any](processor func(seq iter.Seq[V]) R) (
	yield func(V) bool, stop func() R) {

	var (
		next = make(chan V)
		wait = make(chan struct{})
		done bool
		rcvr any
		rslt R
	)

	go func() {
		// recover and propagate panics
		defer func() {
			rcvr = recover()
			done = true
			close(wait)
		}()

		rslt = processor(func(yield func(V) bool) {
			wait <- struct{}{}
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
		if done {
			// maybe panic instead?
			return true
		}

		// yield the next value
		next <- v
		<-wait

		// propapage panics (todo: goexit)
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
		close(next)
		<-wait

		// propapage panics (todo: goexit)
		if rcvr != nil {
			panic(rcvr)
		}
		return rslt
	}

	return yield, stop
}
