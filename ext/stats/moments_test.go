package stats

import (
	"math"
	"testing"
)

func Test_moments(t *testing.T) {
	t.Parallel()

	var s1 moments
	s1.enqueue(1)
	s1.dequeue(1)
	if !math.IsNaN(s1.skewness_pop()) {
		t.Errorf("want NaN")
	}
	if !math.IsNaN(s1.raw_kurtosis_pop()) {
		t.Errorf("want NaN")
	}

	s1.enqueue(+0.5377)
	s1.enqueue(+1.8339)
	s1.enqueue(-2.2588)
	s1.enqueue(+0.8622)
	s1.enqueue(+0.3188)
	s1.enqueue(-1.3077)
	s1.enqueue(-0.4336)
	s1.enqueue(+0.3426)
	s1.enqueue(+3.5784)
	s1.enqueue(+2.7694)

	if got := s1.skewness_pop(); float32(got) != 0.106098293 {
		t.Errorf("got %v, want 0.1061", got)
	}
	if got := s1.skewness_samp(); float32(got) != 0.1258171 {
		t.Errorf("got %v, want 0.1258", got)
	}
	if got := s1.raw_kurtosis_pop(); float32(got) != 2.3121266 {
		t.Errorf("got %v, want 2.3121", got)
	}
	if got := s1.raw_kurtosis_samp(); float32(got) != 2.7482237 {
		t.Errorf("got %v, want 2.7483", got)
	}

	var s2 welford

	s2.enqueue(+0.5377)
	s2.enqueue(+1.8339)
	s2.enqueue(-2.2588)
	s2.enqueue(+0.8622)
	s2.enqueue(+0.3188)
	s2.enqueue(-1.3077)
	s2.enqueue(-0.4336)
	s2.enqueue(+0.3426)
	s2.enqueue(+3.5784)
	s2.enqueue(+2.7694)

	if got, want := s1.mean(), s2.mean(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := s1.stddev_pop(), s2.stddev_pop(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := s1.stddev_samp(), s2.stddev_samp(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	s1.enqueue(math.Pi)
	s1.enqueue(math.Sqrt2)
	s1.enqueue(math.E)
	s1.dequeue(math.Pi)
	s1.dequeue(math.E)
	s1.dequeue(math.Sqrt2)

	if got := s1.skewness_pop(); float32(got) != 0.106098293 {
		t.Errorf("got %v, want 0.1061", got)
	}
	if got := s1.skewness_samp(); float32(got) != 0.1258171 {
		t.Errorf("got %v, want 0.1258", got)
	}
	if got := s1.raw_kurtosis_pop(); float32(got) != 2.3121266 {
		t.Errorf("got %v, want 2.3121", got)
	}
	if got := s1.raw_kurtosis_samp(); float32(got) != 2.7482237 {
		t.Errorf("got %v, want 2.7483", got)
	}
}
