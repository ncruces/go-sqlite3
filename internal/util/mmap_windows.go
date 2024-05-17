//go:build (amd64 || arm64) && !(sqlite3_noshm || sqlite3_nosys)

package util

import (
	"context"

	"github.com/tetratelabs/wazero/experimental"
)

type mmapState struct{}

func withAllocator(ctx context.Context) context.Context {
	return experimental.WithMemoryAllocator(ctx,
		experimental.MemoryAllocatorFunc(newAllocator))
}
