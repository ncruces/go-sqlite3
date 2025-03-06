package seq

import "iter"

// Push takes a consumer function, and returns a yield and a stop function.
// It arranges for the consumer to be called with a Seq iterator.
// The iterator will return all the values passed to the yield function.
// The iterator will stop when the stop function is called.
func iter_Push[V any](consumer func(seq iter.Seq[V])) (
	yield func(V) bool, stop func()) {

	var in V

	coro := func(yieldCoro func(struct{}) bool) {
		seq := func(yieldSeq func(V) bool) {
			for yieldSeq(in) {
				if !yieldCoro(struct{}{}) {
					break
				}
			}
		}
		consumer(seq)
	}

	next, stop := iter.Pull(coro)

	yield = func(v V) bool {
		in = v
		_, more := next()
		if !more {
			stop()
		}
		return more
	}

	return yield, stop
}
