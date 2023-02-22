package compile

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func TestCompile_nil(t *testing.T) {
	_, err := sqlite3.Open(":memory:")
	if err == nil {
		t.Error("want error")
	}
}
