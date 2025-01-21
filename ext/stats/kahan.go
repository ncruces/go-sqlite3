package stats

// https://en.wikipedia.org/wiki/Kahan_summation_algorithm

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
