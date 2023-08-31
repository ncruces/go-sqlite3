package stats

import (
	"math"
	"testing"
)

func Test_welford(t *testing.T) {
	var s1, s2 welford

	s1.enqueue(4)
	s1.enqueue(7)
	s1.enqueue(13)
	s1.enqueue(16)
	if got := s1.average(); got != 10 {
		t.Errorf("got %v, want 10", got)
	}
	if got := s1.var_samp(); got != 30 {
		t.Errorf("got %v, want 30", got)
	}
	if got := s1.var_pop(); got != 22.5 {
		t.Errorf("got %v, want 22.5", got)
	}
	if got := s1.stddev_samp(); got != math.Sqrt(30) {
		t.Errorf("got %v, want √30", got)
	}
	if got := s1.stddev_pop(); got != math.Sqrt(22.5) {
		t.Errorf("got %v, want √22.5", got)
	}

	s1.dequeue(4)
	s2.enqueue(7)
	s2.enqueue(13)
	s2.enqueue(16)
	if s1 != s2 {
		t.Errorf("got %v, want %v", s1, s2)
	}
}
