package util

import (
	"math"
	"testing"
)

func TestUnwrapPointer(t *testing.T) {
	p := Pointer[float64]{Value: math.Pi}
	if got := UnwrapPointer(p); got != math.Pi {
		t.Errorf("want π, got %v", got)
	}
}
