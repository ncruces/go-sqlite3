package sqlite3

import "context"

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
	if r[0] != _OK {
		return s.c.error(r[0])
	}
	return nil
}
