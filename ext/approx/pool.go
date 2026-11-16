package approx

import "sync"

var float64sPool sync.Pool

func getFloat64s(count int) *[]float64 {
	if p, ok := float64sPool.Get().(*[]float64); ok && cap(*p) >= count {
		return p
	}
	w := make([]float64, 0, count)
	return &w
}

func returnFloat64s(p *[]float64) {
	float64sPool.Put(p)
}
