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
	return w.m2.hi / float64(w.n-1) // Bessel's correction
}

func (w welford) stddev_pop() float64 {
	return math.Sqrt(w.var_pop())
}

func (w welford) stddev_samp() float64 {
	return math.Sqrt(w.var_samp())
}

func (w *welford) enqueue(x float64) {
	w.n++
	d1 := x - w.m1.hi - w.m1.lo
	w.m1.add(d1 / float64(w.n))
	d2 := x - w.m1.hi - w.m1.lo
	w.m2.add(d1 * d2)
}

func (w *welford) dequeue(x float64) {
	w.n--
	d1 := x - w.m1.hi - w.m1.lo
	w.m1.sub(d1 / float64(w.n))
	d2 := x - w.m1.hi - w.m1.lo
	w.m2.sub(d1 * d2)
}

type welford2 struct {
	x, y, c kahan
	n       uint64
}

func (w welford2) covar_pop() float64 {
	return w.c.hi / float64(w.n)
}

func (w welford2) covar_samp() float64 {
	return w.c.hi / float64(w.n-1) // Bessel's correction
}

func (w *welford2) enqueue(x, y float64) {
	w.n++
	dx := x - w.x.hi - w.x.lo
	dy := y - w.y.hi - w.y.lo
	w.x.add(dx / float64(w.n))
	w.y.add(dy / float64(w.n))
	d2 := y - w.y.hi - w.y.lo
	w.c.add(dx * d2)
}

func (w *welford2) dequeue(x, y float64) {
	w.n--
	dx := x - w.x.hi - w.x.lo
	dy := y - w.y.hi - w.y.lo
	w.x.sub(dx / float64(w.n))
	w.y.sub(dy / float64(w.n))
	d2 := y - w.y.hi - w.y.lo
	w.c.sub(dx * d2)
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
