package sqlite3

import (
	"context"
	"math"
)

type Stmt struct {
	c      *Conn
	handle uint32
}

func (s *Stmt) Close() error {
	r, err := s.c.api.finalize.Call(context.TODO(), uint64(s.handle))
	if err != nil {
		return err
	}

	s.handle = 0
	return s.c.error(r[0])
}

func (s *Stmt) BindBool(param int, value bool) error {
	if value {
		return s.BindInt64(param, 1)
	}
	return s.BindInt64(param, 0)
}

func (s *Stmt) BindInt(param int, value int) error {
	return s.BindInt64(param, int64(value))
}

func (s *Stmt) BindInt64(param int, value int64) error {
	r, err := s.c.api.bindInteger.Call(context.TODO(),
		uint64(s.handle), uint64(param), uint64(value))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindFloat(param int, value float64) error {
	r, err := s.c.api.bindFloat.Call(context.TODO(),
		uint64(s.handle), uint64(param), math.Float64bits(value))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindText(param int, value string) error {
	ptr := s.c.newString(value)
	r, err := s.c.api.bindText.Call(context.TODO(),
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)),
		s.c.api.destructor, _UTF8)
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindBlob(param int, value []byte) error {
	ptr := s.c.newBytes(value)
	r, err := s.c.api.bindBlob.Call(context.TODO(),
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)),
		s.c.api.destructor)
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindNull(param int) error {
	r, err := s.c.api.bindNull.Call(context.TODO(),
		uint64(s.handle), uint64(param))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}
