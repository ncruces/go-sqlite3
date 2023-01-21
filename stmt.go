package sqlite3

import (
	"math"
)

type Stmt struct {
	c      *Conn
	handle uint32
	err    error
}

func (s *Stmt) Close() error {
	r, err := s.c.api.finalize.Call(s.c.ctx, uint64(s.handle))
	if err != nil {
		return err
	}

	s.handle = 0
	return s.c.error(r[0])
}

func (s *Stmt) Reset() error {
	r, err := s.c.api.reset.Call(s.c.ctx, uint64(s.handle))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) Step() bool {
	r, err := s.c.api.step.Call(s.c.ctx, uint64(s.handle))
	if err != nil {
		s.err = err
		return false
	}
	if r[0] == _ROW {
		return true
	}
	if r[0] == _DONE {
		s.err = nil
	} else {
		s.err = s.c.error(r[0])
	}
	return false
}

func (s *Stmt) Err() error {
	return s.err
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
	r, err := s.c.api.bindInteger.Call(s.c.ctx,
		uint64(s.handle), uint64(param), uint64(value))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindFloat(param int, value float64) error {
	r, err := s.c.api.bindFloat.Call(s.c.ctx,
		uint64(s.handle), uint64(param), math.Float64bits(value))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindText(param int, value string) error {
	ptr := s.c.newString(value)
	r, err := s.c.api.bindText.Call(s.c.ctx,
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
	r, err := s.c.api.bindBlob.Call(s.c.ctx,
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)),
		s.c.api.destructor)
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) BindNull(param int) error {
	r, err := s.c.api.bindNull.Call(s.c.ctx,
		uint64(s.handle), uint64(param))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

func (s *Stmt) ColumnBool(col int) bool {
	if i := s.ColumnInt64(col); i != 0 {
		return true
	}
	return false
}

func (s *Stmt) ColumnInt(col int) int {
	return int(s.ColumnInt64(col))
}

func (s *Stmt) ColumnInt64(col int) int64 {
	r, err := s.c.api.columnInteger.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}
	return int64(r[0])
}

func (s *Stmt) ColumnFloat(col int) float64 {
	r, err := s.c.api.columnInteger.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}
	return math.Float64frombits(r[0])
}

func (s *Stmt) ColumnText(col int) string {
	r, err := s.c.api.columnText.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}

	ptr := uint32(r[0])
	if ptr == 0 {
		r, err = s.c.api.errcode.Call(s.c.ctx, uint64(s.handle))
		if err != nil {
			panic(err)
		}
		s.err = s.c.error(r[0])
		return ""
	}

	r, err = s.c.api.columnBytes.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}

	mem, ok := s.c.memory.Read(ptr, uint32(r[0]))
	if !ok {
		panic(rangeErr)
	}
	return string(mem)
}

func (s *Stmt) ColumnBlob(col int, buf []byte) []byte {
	r, err := s.c.api.columnBlob.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}

	ptr := uint32(r[0])
	if ptr == 0 {
		r, err = s.c.api.errcode.Call(s.c.ctx, uint64(s.handle))
		if err != nil {
			panic(err)
		}
		s.err = s.c.error(r[0])
		return nil
	}

	r, err = s.c.api.columnBytes.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}

	mem, ok := s.c.memory.Read(ptr, uint32(r[0]))
	if !ok {
		panic(rangeErr)
	}
	return append(buf[0:0], mem...)
}
