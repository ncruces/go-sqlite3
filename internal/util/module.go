package util

import (
	"context"

	"github.com/tetratelabs/wazero/experimental"

	"github.com/ncruces/go-sqlite3/internal/alloc"
)

type ConnKey struct{}

type moduleKey struct{}
type moduleState struct {
	sysError error
	mmapState
	handleState
}

func NewContext(ctx context.Context) context.Context {
	state := new(moduleState)
	ctx = experimental.WithMemoryAllocator(ctx, experimental.MemoryAllocatorFunc(alloc.NewMemory))
	ctx = experimental.WithCloseNotifier(ctx, state)
	ctx = context.WithValue(ctx, moduleKey{}, state)
	return ctx
}

func GetSystemError(ctx context.Context) error {
	s := ctx.Value(moduleKey{}).(*moduleState)
	return s.sysError
}

func SetSystemError(ctx context.Context, err error) {
	s, ok := ctx.Value(moduleKey{}).(*moduleState)
	if ok {
		s.sysError = err
	}
}
