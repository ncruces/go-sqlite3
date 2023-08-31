package stats

import "math"

// Welford's algorithm with Kahan summation:
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Welford's_online_algorithm
// https://en.wikipedia.org/wiki/Kahan_summation_algorithm

type welford struct {
	m1, m2 kahan
	n      uint64
}

func (w welford) average() float64 {
	return w.m1.hi
}

func (w welford) var_pop() float64 {
	return w.m2.hi / float64(w.n)
}

func (w welford) var_samp() float64 {
	return w.m2.hi / float64(w.n-1)
}

func (w welford) stddev_pop() float64 {
	return math.Sqrt(w.var_pop())
}

func (w welford) stddev_samp() float64 {
	return math.Sqrt(w.var_samp())
}

func (w *welford) enqueue(x float64) {
	w.n++
	d1 := x - w.m1.hi
	w.m1.add(d1 / float64(w.n))
	d2 := x - w.m1.hi
	w.m2.add(d1 * d2)
}

func (w *welford) dequeue(x float64) {
	w.n--
	d1 := x - w.m1.hi
	w.m1.sub(d1 / float64(w.n))
	d2 := x - w.m1.hi
	w.m2.sub(d1 * d2)
}

type kahan struct{ hi, lo float64 }

func (k *kahan) add(x float64) {
	y := k.lo + x
	t := k.hi + y
	k.lo = y - (t - k.hi)
	k.hi = t
}

func (k *kahan) sub(x float64) {
	y := k.lo - x
	t := k.hi + y
	k.lo = y - (t - k.hi)
	k.hi = t
}
