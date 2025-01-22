package stats

import "math"

// Fisher–Pearson skewness and kurtosis using
// Terriberry's algorithm with Kahan summation:
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Higher-order_statistics

type moments struct {
	m1, m2, m3, m4 kahan
	n              int64
}

func (m moments) mean() float64 {
	return m.m1.hi
}

func (m moments) var_pop() float64 {
	return m.m2.hi / float64(m.n)
}

func (m moments) var_samp() float64 {
	return m.m2.hi / float64(m.n-1) // Bessel's correction
}

func (m moments) stddev_pop() float64 {
	return math.Sqrt(m.var_pop())
}

func (m moments) stddev_samp() float64 {
	return math.Sqrt(m.var_samp())
}

func (m moments) skewness_pop() float64 {
	m2 := m.m2.hi
	if div := m2 * m2 * m2; div != 0 {
		return m.m3.hi * math.Sqrt(float64(m.n)/div)
	}
	return math.NaN()
}

func (m moments) skewness_samp() float64 {
	n := m.n
	// https://mathworks.com/help/stats/skewness.html#f1132178
	return m.skewness_pop() * math.Sqrt(float64(n*(n-1))) / float64(n-2)
}

func (m moments) kurtosis_pop() float64 {
	return m.raw_kurtosis_pop() - 3
}

func (m moments) raw_kurtosis_pop() float64 {
	m2 := m.m2.hi
	if div := m2 * m2; div != 0 {
		return m.m4.hi * float64(m.n) / div
	}
	return math.NaN()
}

func (m moments) kurtosis_samp() float64 {
	n := m.n
	k := math.FMA(m.raw_kurtosis_pop(), float64(n+1), float64(3-3*n))
	return k * float64(n-1) / float64((n-2)*(n-3))
}

func (m moments) raw_kurtosis_samp() float64 {
	n := m.n
	// https://mathworks.com/help/stats/kurtosis.html#f4975293
	k := math.FMA(m.raw_kurtosis_pop(), float64(n+1), float64(3-3*n))
	return math.FMA(k, float64(n-1)/float64((n-2)*(n-3)), 3)
}

func (m *moments) enqueue(x float64) {
	n := m.n + 1
	m.n = n
	d1 := x - m.m1.hi - m.m1.lo
	dn := d1 / float64(n)
	d2 := dn * dn
	t1 := d1 * dn * float64(n-1)
	m.m4.add(t1*d2*float64(n*n-3*n+3) + 6*d2*m.m2.hi - 4*dn*m.m3.hi)
	m.m3.add(t1*dn*float64(n-2) - 3*dn*m.m2.hi)
	m.m2.add(t1)
	m.m1.add(dn)
}

func (m *moments) dequeue(x float64) {
	n := m.n - 1
	if n <= 0 {
		*m = moments{}
		return
	}
	m.n = n
	d1 := x - m.m1.hi - m.m1.lo
	dn := d1 / float64(n)
	d2 := dn * dn
	t1 := d1 * dn * float64(n+1)
	m.m4.sub(t1*d2*float64(n*n+3*n+3) - 6*d2*m.m2.hi - 4*dn*m.m3.hi)
	m.m3.sub(t1*dn*float64(n+2) - 3*dn*m.m2.hi)
	m.m2.sub(t1)
	m.m1.sub(dn)
}
