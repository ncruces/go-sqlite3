//go:build !(unix || windows) || sqlite3_nosys

package util

import "context"

func withAllocator(ctx context.Context) context.Context {
	return ctx
}
