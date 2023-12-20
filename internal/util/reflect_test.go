package util

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestReflectType(t *testing.T) {
	tests := []any{nil, 1, math.Pi, "abc"}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt), func(t *testing.T) {
			want := fmt.Sprintf("%T", tt)
			got := fmt.Sprintf("%v", ReflectType(reflect.ValueOf(tt)))
			if got != want {
				t.Errorf("ReflectType() = %v, want %v", got, want)
			}
		})
	}
}
