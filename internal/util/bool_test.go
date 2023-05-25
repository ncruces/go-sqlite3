package util

import "testing"

func TestParseBool(t *testing.T) {
	tests := []struct {
		str string
		val bool
		ok  bool
	}{
		{"", false, false},
		{"0", false, true},
		{"1", true, true},
		{"9", true, true},
		{"T", false, false},
		{"true", true, true},
		{"FALSE", false, true},
		{"false?", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			gotVal, gotOK := ParseBool(tt.str)
			if gotVal != tt.val || gotOK != tt.ok {
				t.Errorf("ParseBool(%q) = (%v, %v) want (%v, %v)", tt.str, gotVal, gotOK, tt.val, tt.ok)
			}
		})
	}
}
