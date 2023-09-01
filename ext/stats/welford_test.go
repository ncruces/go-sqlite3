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
	s1.m1.lo, s2.m1.lo = 0, 0
	s1.m2.lo, s2.m2.lo = 0, 0
	if s1 != s2 {
		t.Errorf("got %v, want %v", s1, s2)
	}
}

func Test_covar(t *testing.T) {
	var c1, c2 welford2

	c1.enqueue(3, 70)
	c1.enqueue(5, 80)
	c1.enqueue(2, 60)
	c1.enqueue(7, 90)
	c1.enqueue(4, 75)

	if got := c1.covar_samp(); got != 21.25 {
		t.Errorf("got %v, want 21.25", got)
	}
	if got := c1.covar_pop(); got != 17 {
		t.Errorf("got %v, want 17", got)
	}

	c1.dequeue(3, 70)
	c2.enqueue(5, 80)
	c2.enqueue(2, 60)
	c2.enqueue(7, 90)
	c2.enqueue(4, 75)
	c1.x.lo, c2.x.lo = 0, 0
	c1.y.lo, c2.y.lo = 0, 0
	c1.c.lo, c2.c.lo = 0, 0
	if c1 != c2 {
		t.Errorf("got %v, want %v", c1, c2)
	}
}
