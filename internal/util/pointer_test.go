package util_test

import (
	"math"
	"testing"

	"github.com/ncruces/go-sqlite3/internal/util"
)

func TestUnwrapPointer(t *testing.T) {
	p := util.Pointer[float64]{Value: math.Pi}
	if got := util.UnwrapPointer(p); got != math.Pi {
		t.Errorf("want π, got %v", got)
	}
}
