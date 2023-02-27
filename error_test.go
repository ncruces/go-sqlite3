package sqlite3

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func Test_assertErr(t *testing.T) {
	err := assertErr()
	if s := err.Error(); !strings.HasPrefix(s, "sqlite3: assertion failed") || !strings.HasSuffix(s, "error_test.go:11)") {
		t.Errorf("got %q", s)
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	err := Error{code: 0x8080}
	if rc := err.Code(); rc != 0x80 {
		t.Errorf("got %#x, want 0x80", rc)
	}
	if !errors.Is(&err, ErrorCode(0x80)) {
		t.Errorf("want true")
	}
	if rc := err.ExtendedCode(); rc != 0x8080 {
		t.Errorf("got %#x, want 0x8080", rc)
	}
	if !errors.Is(&err, ExtendedErrorCode(0x8080)) {
		t.Errorf("want true")
	}
	if s := err.Error(); s != "sqlite3: 32896" {
		t.Errorf("got %q", s)
	}
	if !errors.Is(err.ExtendedCode(), ErrorCode(0x80)) {
		t.Errorf("want true")
	}
}

func TestError_Temporary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code uint64
		want bool
	}{
		{"ERROR", uint64(ERROR), false},
		{"BUSY", uint64(BUSY), true},
		{"BUSY_RECOVERY", uint64(BUSY_RECOVERY), true},
		{"BUSY_SNAPSHOT", uint64(BUSY_SNAPSHOT), true},
		{"BUSY_TIMEOUT", uint64(BUSY_TIMEOUT), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			{
				err := &Error{code: tt.code}
				if got := err.Temporary(); got != tt.want {
					t.Errorf("Error.Temporary(%d) = %v, want %v", tt.code, got, tt.want)
				}
			}
			{
				err := ErrorCode(tt.code)
				if got := err.Temporary(); got != tt.want {
					t.Errorf("ErrorCode.Temporary(%d) = %v, want %v", tt.code, got, tt.want)
				}
			}
			{
				err := ExtendedErrorCode(tt.code)
				if got := err.Temporary(); got != tt.want {
					t.Errorf("ExtendedErrorCode.Temporary(%d) = %v, want %v", tt.code, got, tt.want)
				}
			}
		})
	}
}

func TestError_Timeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code uint64
		want bool
	}{
		{"ERROR", uint64(ERROR), false},
		{"BUSY", uint64(BUSY), false},
		{"BUSY_RECOVERY", uint64(BUSY_RECOVERY), false},
		{"BUSY_SNAPSHOT", uint64(BUSY_SNAPSHOT), false},
		{"BUSY_TIMEOUT", uint64(BUSY_TIMEOUT), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			{
				err := &Error{code: tt.code}
				if got := err.Timeout(); got != tt.want {
					t.Errorf("Error.Timeout(%d) = %v, want %v", tt.code, got, tt.want)
				}
			}
			{
				err := ExtendedErrorCode(tt.code)
				if got := err.Timeout(); got != tt.want {
					t.Errorf("Error.Timeout(%d) = %v, want %v", tt.code, got, tt.want)
				}
			}
		})
	}
}

func Test_ErrorCode_Error(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Test all error codes.
	for i := 0; i == int(ErrorCode(i)); i++ {
		want := "sqlite3: "
		r, _ := db.api.errstr.Call(context.TODO(), uint64(i))
		if r != nil {
			want += db.mem.readString(uint32(r[0]), _MAX_STRING)
		}

		got := ErrorCode(i).Error()
		if got != want {
			t.Fatalf("got %q, want %q, with %d", got, want, i)
		}
	}
}

func Test_ExtendedErrorCode_Error(t *testing.T) {
	t.Parallel()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Test all extended error codes.
	for i := 0; i == int(ExtendedErrorCode(i)); i++ {
		want := "sqlite3: "
		r, _ := db.api.errstr.Call(context.TODO(), uint64(i))
		if r != nil {
			want += db.mem.readString(uint32(r[0]), _MAX_STRING)
		}

		got := ExtendedErrorCode(i).Error()
		if got != want {
			t.Fatalf("got %q, want %q, with %d", got, want, i)
		}
	}
}
