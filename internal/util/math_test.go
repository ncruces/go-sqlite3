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

func TestFloatIsInt(t *testing.T) {
	tests := []struct {
		arg  float64
		want int64
		ok   bool
	}{
		{0, 0, true},
		{math.Copysign(0, -1), 0, true},
		{1, 1, true},
		{-1, -1, true},
		{1.5, 0, false},
		{math.MinInt64, math.MinInt64, true},
		{9223372036854774784, 9223372036854774784, true},
		{9223372036854775808, 0, false},
		{math.Inf(1), 0, false},
		{math.Inf(-1), 0, false},
		{math.NaN(), 0, false},
	}
	for _, tt := range tests {
		got, ok := FloatIsInt(tt.arg)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("FloatIsInt(%v) = (%d, %v), want (%d, %v)", tt.arg, got, ok, tt.want, tt.ok)
		}
	}
}

func TestAppendFloatNumber(t *testing.T) {
	tests := []struct {
		arg     float64
		bits    int
		wantFlt string
		wantNum string
	}{
		{0, 64, "0.0", "0"},
		{math.Copysign(0, -1), 64, "-0.0", "0"},
		{1, 64, "1.0", "1"},
		{1.5, 64, "1.5", "1.5"},
		{1e6, 64, "1e+06", "1000000"},
		{9007199254740992, 64, "9.007199254740992e+15", "9007199254740992"},
		{9223372036854775808, 64, "9.223372036854776e+18", "9.223372036854776e+18"},
		{float64(float32(0.1)), 32, "0.1", "0.1"},
		{math.Inf(1), 64, "9.0e999", "9.0e999"},
		{math.Inf(-1), 64, "-9.0e999", "-9.0e999"},
		{math.NaN(), 64, "null", "null"},
	}
	for _, tt := range tests {
		if got := string(AppendFloat([]byte("x:"), tt.arg, tt.bits)); got != "x:"+tt.wantFlt {
			t.Errorf("AppendFloat(%v, %d) = %q, want %q", tt.arg, tt.bits, got, "x:"+tt.wantFlt)
		}
		if got := string(AppendNumber([]byte("x:"), tt.arg, tt.bits)); got != "x:"+tt.wantNum {
			t.Errorf("AppendNumber(%v, %d) = %q, want %q", tt.arg, tt.bits, got, "x:"+tt.wantNum)
		}
	}
}
