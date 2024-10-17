package util

import (
	"math"
	"testing"
)

func Test_abs(t *testing.T) {
	tests := []struct {
		arg  int
		want int
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{math.MaxInt, math.MaxInt},
		{math.MinInt, math.MinInt},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := abs(tt.arg); got != tt.want {
				t.Errorf("abs(%d) = %d, want %d", tt.arg, got, tt.want)
			}
		})
	}
}

func Test_GCD(t *testing.T) {
	tests := []struct {
		arg1 int
		arg2 int
		want int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{1, 0, 1},
		{1, 1, 1},
		{2, 3, 1},
		{42, 56, 14},
		{48, -18, 6},
		{1e9, 1e9, 1e9},
		{1e9, -1e9, 1e9},
		{-1e9, -1e9, 1e9},
		{math.MaxInt, math.MaxInt, math.MaxInt},
		{math.MinInt, math.MinInt, math.MinInt},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := GCD(tt.arg1, tt.arg2); got != tt.want {
				t.Errorf("gcd(%d, %d) = %d, want %d", tt.arg1, tt.arg2, got, tt.want)
			}
		})
	}
}

func Test_LCM(t *testing.T) {
	tests := []struct {
		arg1 int
		arg2 int
		want int
	}{
		{0, 0, 0},
		{0, 1, 0},
		{1, 0, 0},
		{1, 1, 1},
		{2, 3, 6},
		{42, 56, 168},
		{48, -18, 144},
		{1e9, 1e9, 1e9},
		{1e9, -1e9, 1e9},
		{-1e9, -1e9, 1e9},
		{math.MaxInt, math.MaxInt, math.MaxInt},
		{math.MinInt, math.MinInt, math.MinInt},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := LCM(tt.arg1, tt.arg2); got != tt.want {
				t.Errorf("lcm(%d, %d) = %d, want %d", tt.arg1, tt.arg2, got, tt.want)
			}
		})
	}
}
