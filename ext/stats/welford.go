package stats

import (
	"math"
	"strconv"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Welford's algorithm with Kahan summation:
// The effect of truncation in statistical computation [van Reeken, AJ 1970]
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Welford's_online_algorithm

type welford struct {
	m1, m2 kahan
	n      int64
}

func (w welford) mean() float64 {
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
	n := w.n + 1
	w.n = n
	d1 := x - w.m1.hi - w.m1.lo
	w.m1.add(d1 / float64(n))
	d2 := x - w.m1.hi - w.m1.lo
	w.m2.add(d1 * d2)
}

func (w *welford) dequeue(x float64) {
	n := w.n - 1
	if n <= 0 {
		*w = welford{}
		return
	}
	w.n = n
	d1 := x - w.m1.hi - w.m1.lo
	w.m1.sub(d1 / float64(n))
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

func (w welford2) regr_json(dst []byte) []byte {
	dst = append(dst, `{"count":`...)
	dst = strconv.AppendInt(dst, w.regr_count(), 10)
	dst = append(dst, `,"avgy":`...)
	dst = util.AppendNumber(dst, w.regr_avgy())
	dst = append(dst, `,"avgx":`...)
	dst = util.AppendNumber(dst, w.regr_avgx())
	dst = append(dst, `,"syy":`...)
	dst = util.AppendNumber(dst, w.regr_syy())
	dst = append(dst, `,"sxx":`...)
	dst = util.AppendNumber(dst, w.regr_sxx())
	dst = append(dst, `,"sxy":`...)
	dst = util.AppendNumber(dst, w.regr_sxy())
	dst = append(dst, `,"slope":`...)
	dst = util.AppendNumber(dst, w.regr_slope())
	dst = append(dst, `,"intercept":`...)
	dst = util.AppendNumber(dst, w.regr_intercept())
	dst = append(dst, `,"r2":`...)
	dst = util.AppendNumber(dst, w.regr_r2())
	return append(dst, '}')
}

func (w *welford2) enqueue(y, x float64) {
	n := w.n + 1
	w.n = n
	d1y := y - w.m1y.hi - w.m1y.lo
	d1x := x - w.m1x.hi - w.m1x.lo
	w.m1y.add(d1y / float64(n))
	w.m1x.add(d1x / float64(n))
	d2y := y - w.m1y.hi - w.m1y.lo
	d2x := x - w.m1x.hi - w.m1x.lo
	w.m2y.add(d1y * d2y)
	w.m2x.add(d1x * d2x)
	w.cov.add(d1y * d2x)
}

func (w *welford2) dequeue(y, x float64) {
	n := w.n - 1
	if n <= 0 {
		*w = welford2{}
		return
	}
	w.n = n
	d1y := y - w.m1y.hi - w.m1y.lo
	d1x := x - w.m1x.hi - w.m1x.lo
	w.m1y.sub(d1y / float64(n))
	w.m1x.sub(d1x / float64(n))
	d2y := y - w.m1y.hi - w.m1y.lo
	d2x := x - w.m1x.hi - w.m1x.lo
	w.m2y.sub(d1y * d2y)
	w.m2x.sub(d1x * d2x)
	w.cov.sub(d1y * d2x)
}
