package vfs

import (
	"math"
	"testing"
)

func Test_ErrorCode_Error(t *testing.T) {
	tests := []struct {
		code _ErrorCode
		want string
	}{
		{_OK, "sqlite3: not an error"},
		{_ERROR, "sqlite3: SQL logic error"},
		{math.MaxUint32, "sqlite3: unknown error"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.code.Error(); got != tt.want {
				t.Errorf("_ErrorCode.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
