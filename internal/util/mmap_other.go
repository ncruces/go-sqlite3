//go:build !(darwin || linux || windows) || !(amd64 || arm64 || riscv64) || sqlite3_noshm || sqlite3_nosys

package util

import "context"

type mmapState struct{}

func withAllocator(ctx context.Context) context.Context {
	return ctx
}
