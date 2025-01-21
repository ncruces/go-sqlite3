package stats

import (
	"math"
	"testing"
)

func Test_welford(t *testing.T) {
	t.Parallel()

	var s1, s2 welford
	s1.enqueue(1)
	s1.dequeue(1)

	s1.enqueue(4)
	s1.enqueue(7)
	s1.enqueue(13)
	s1.enqueue(16)
	if got := s1.mean(); got != 10 {
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
	if s1.var_pop() != s2.var_pop() {
		t.Errorf("got %v, want %v", s1, s2)
	}
}

func Test_covar(t *testing.T) {
	t.Parallel()

	var c1, c2 welford2
	c1.enqueue(1, 1)
	c1.dequeue(1, 1)

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
	if c1.covar_pop() != c2.covar_pop() {
		t.Errorf("got %v, want %v", c1.covar_pop(), c2.covar_pop())
	}
}

func Test_correlation(t *testing.T) {
	t.Parallel()

	var c welford2
	c.enqueue(1, 3)
	c.enqueue(2, 2)
	c.enqueue(3, 1)

	if got := c.correlation(); got != -1 {
		t.Errorf("got %v, want -1", got)
	}
}
