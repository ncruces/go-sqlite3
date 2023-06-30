package util

import (
	"context"
	"io"
)

type handleKey struct{}
type handleState struct {
	handles []any
}

func NewContext(ctx context.Context) (context.Context, io.Closer) {
	state := new(handleState)
	return context.WithValue(ctx, handleKey{}, state), state
}

func (s *handleState) Close() (err error) {
	for _, h := range s.handles {
		if c, ok := h.(io.Closer); ok {
			if e := c.Close(); err == nil {
				err = e
			}
		}
	}
	s.handles = nil
	return err
}

func GetHandle(ctx context.Context, id uint32) any {
	s := ctx.Value(handleKey{}).(*handleState)
	return s.handles[id]
}

func DelHandle(ctx context.Context, id uint32) error {
	s := ctx.Value(handleKey{}).(*handleState)
	a := s.handles[id]
	s.handles[id] = nil
	if c, ok := a.(io.Closer); ok {
		return c.Close()
	}
	return nil

}

func AddHandle(ctx context.Context, a any) (id uint32) {
	s := ctx.Value(handleKey{}).(*handleState)

	// Find an empty slot.
	for id, h := range s.handles {
		if h == nil {
			s.handles[id] = a
			return uint32(id)
		}
	}

	// Add a new slot.
	s.handles = append(s.handles, a)
	return uint32(len(s.handles) - 1)
}
