package testutil

import (
	"context"
	"testing"

	"github.com/ncruces/go-sqlite3"
)

func Context(t testing.TB) context.Context {
	return sqlite3.WithMaxMemory(t.Context(), 32*1024*1024)
}
