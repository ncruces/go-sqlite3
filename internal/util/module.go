package util

import "context"

type ConnKey struct{}

type moduleKey struct{}
type moduleState struct {
	sysError error
	mmapState
	handleState
}

func NewContext(ctx context.Context) (context.Context, context.CancelFunc) {
	state := new(moduleState)
	ctx = context.WithValue(ctx, moduleKey{}, state)
	return ctx, state.Close
}

func GetSystemError(ctx context.Context) error {
	// Test needed to simplify testing.
	s, ok := ctx.Value(moduleKey{}).(*moduleState)
	if ok {
		return s.sysError
	}
	return nil
}

func SetSystemError(ctx context.Context, err error) {
	// Test needed to simplify testing.
	s, ok := ctx.Value(moduleKey{}).(*moduleState)
	if ok {
		s.sysError = err
	}
}
