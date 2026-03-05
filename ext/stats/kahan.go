package stats

// https://en.wikipedia.org/wiki/Kahan_summation_algorithm

type kahan struct{ hi, lo float64 }

func (k *kahan) add(x float64) {
	y := float64(k.lo + x)
	t := float64(k.hi + y)
	k.lo = y - float64(t-k.hi)
	k.hi = t
}

func (k *kahan) sub(x float64) {
	y := float64(k.lo - x)
	t := float64(k.hi + y)
	k.lo = y - float64(t-k.hi)
	k.hi = t
}
