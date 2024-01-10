package stats

import "math"

// Welford's algorithm with Kahan summation:
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Welford's_online_algorithm
// https://en.wikipedia.org/wiki/Kahan_summation_algorithm

// See also:
// https://duckdb.org/docs/sql/aggregates.html#statistical-aggregates

type welford struct {
	m1, m2 kahan
	n      int64
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
	m1y, m2y kahan
	m1x, m2x kahan
	cov      kahan
	n        int64
}

func (w welford2) covar_pop() float64 {
	return w.cov.hi / float64(w.n)
}

func (w welford2) covar_samp() float64 {
	return w.cov.hi / float64(w.n-1) // Bessel's correction
}

func (w welford2) correlation() float64 {
	return w.cov.hi / math.Sqrt(w.m2y.hi*w.m2x.hi)
}

func (w welford2) regr_avgy() float64 {
	return w.m1y.hi
}

func (w welford2) regr_avgx() float64 {
	return w.m1x.hi
}

func (w welford2) regr_syy() float64 {
	return w.m2y.hi
}

func (w welford2) regr_sxx() float64 {
	return w.m2x.hi
}

func (w welford2) regr_sxy() float64 {
	return w.cov.hi
}

func (w welford2) regr_count() int64 {
	return w.n
}

func (w welford2) regr_slope() float64 {
	return w.cov.hi / w.m2x.hi
}

func (w welford2) regr_intercept() float64 {
	slope := -w.regr_slope()
	hi := math.FMA(slope, w.m1x.hi, w.m1y.hi)
	lo := math.FMA(slope, w.m1x.lo, w.m1y.lo)
	return hi + lo
}

func (w welford2) regr_r2() float64 {
	return w.cov.hi * w.cov.hi / (w.m2y.hi * w.m2x.hi)
}

func (w *welford2) enqueue(y, x float64) {
	w.n++
	d1y := y - w.m1y.hi - w.m1y.lo
	d1x := x - w.m1x.hi - w.m1x.lo
	w.m1y.add(d1y / float64(w.n))
	w.m1x.add(d1x / float64(w.n))
	d2y := y - w.m1y.hi - w.m1y.lo
	d2x := x - w.m1x.hi - w.m1x.lo
	w.m2y.add(d1y * d2y)
	w.m2x.add(d1x * d2x)
	w.cov.add(d1y * d2x)
}

func (w *welford2) dequeue(y, x float64) {
	w.n--
	d1y := y - w.m1y.hi - w.m1y.lo
	d1x := x - w.m1x.hi - w.m1x.lo
	w.m1y.sub(d1y / float64(w.n))
	w.m1x.sub(d1x / float64(w.n))
	d2y := y - w.m1y.hi - w.m1y.lo
	d2x := x - w.m1x.hi - w.m1x.lo
	w.m2y.sub(d1y * d2y)
	w.m2x.sub(d1x * d2x)
	w.cov.sub(d1y * d2x)
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
