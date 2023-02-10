package sqlite3

import (
	"strings"
	"testing"
)

func TestError(t *testing.T) {
	err := Error{code: 0x8080}
	if rc := err.Code(); rc != 0x80 {
		t.Errorf("got %#x, want 0x80", rc)
	}
	if rc := err.ExtendedCode(); rc != 0x8080 {
		t.Errorf("got %#x, want 0x8080", rc)
	}
	if s := err.Error(); s != "sqlite3: 32896" {
		t.Errorf("got %q", s)
	}
}

func Test_assertErr(t *testing.T) {
	err := assertErr()
	if s := err.Error(); !strings.HasPrefix(s, "sqlite3: assertion failed") || !strings.HasSuffix(s, "error_test.go:22)") {
		t.Errorf("got %q", s)
	}
}
