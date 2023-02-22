package compile

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func TestCompile_empty(t *testing.T) {
	sqlite3.Binary = []byte("\x00asm\x01\x00\x00\x00")
	_, err := sqlite3.Open(":memory:")
	if err == nil {
		t.Error("want error")
	}
}
