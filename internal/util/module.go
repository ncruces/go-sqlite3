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

func NewContext(ctx context.Context) context.Context {
	state := new(moduleState)
	ctx = experimental.WithCloseNotifier(ctx, state)
	ctx = context.WithValue(ctx, moduleKey{}, state)
	return ctx
}

func (s *moduleState) CloseNotify(ctx context.Context, exitCode uint32) {
	s.handleState.closeNotify()
	s.mmapState.closeNotify()
}
