//go:build !coro

package seq

import (
	"iter"
	"runtime"
)

// Push takes a consumer function, and returns a yield and a stop function.
// It arranges for the consumer to be called with a Seq iterator.
// The iterator will return all the values passed to the yield function.
// The iterator will stop when the stop function is called.
func iter_Push[V any](consumer func(seq iter.Seq[V])) (
	yield func(V) bool, stop func()) {

	var (
		v          V
		done       bool
		panicValue any
		seqDone    bool // to detect Goexit
		swtch      = make(chan struct{})
	)

	go func() {
		// Recover and propagate panics from consumer.
		defer func() {
			if p := recover(); p != nil {
				panicValue = p
			} else if !seqDone {
				panicValue = goexitPanicValue
			}
			done = true
			close(swtch)
		}()

		<-swtch
		consumer(func(yield func(V) bool) {
			for !done {
				if !yield(v) {
					break
				}
				swtch <- struct{}{}
				<-swtch
			}
		})
		seqDone = true
	}()

	yield = func(v1 V) bool {
		v = v1
		// Yield the next value.
		// Panics if stop has been called.
		swtch <- struct{}{}
		<-swtch

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
		done = true
		// Finish the iteration.
		// Panics if stop has been called.
		swtch <- struct{}{}
		<-swtch

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
