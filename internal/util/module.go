package util

import (
	"context"

	"github.com/tetratelabs/wazero/experimental"
)

type moduleKey struct{}
type moduleState struct {
	handleState
	mmapState
}

func NewContext(ctx context.Context, mappableMemory bool) context.Context {
	state := new(moduleState)
	state.mmapState.enabled = mappableMemory
	ctx = context.WithValue(ctx, moduleKey{}, state)
	ctx = experimental.WithCloseNotifier(ctx, state)
	ctx = experimental.WithMemoryAllocator(ctx, mappableMemoryAllocator{})
	return ctx
}

func (s *moduleState) CloseNotify(ctx context.Context, exitCode uint32) {
	s.handleState.closeNotify()
	s.mmapState.closeNotify()
}
