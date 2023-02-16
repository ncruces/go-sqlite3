package sqlite3

import (
	"math"
)

// Stmt is a prepared statement object.
//
// https://www.sqlite.org/c3ref/stmt.html
type Stmt struct {
	c      *Conn
	handle uint32
	err    error
}

// Close destroys the prepared statement object.
//
// https://www.sqlite.org/c3ref/finalize.html
func (s *Stmt) Close() error {
	if s == nil {
		return nil
	}

	r, err := s.c.api.finalize.Call(s.c.ctx, uint64(s.handle))
	if err != nil {
		return err
	}

	s.handle = 0
	return s.c.error(r[0])
}

// Reset resets the prepared statement object.
//
// https://www.sqlite.org/c3ref/reset.html
func (s *Stmt) Reset() error {
	r, err := s.c.api.reset.Call(s.c.ctx, uint64(s.handle))
	if err != nil {
		return err
	}
	s.err = nil
	return s.c.error(r[0])
}

// ClearBindings resets all bindings on the prepared statement.
//
// https://www.sqlite.org/c3ref/clear_bindings.html
func (s *Stmt) ClearBindings() error {
	r, err := s.c.api.clearBindings.Call(s.c.ctx, uint64(s.handle))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

// Step evaluates the SQL statement.
// If the SQL statement being executed returns any data,
// then true is returned each time a new row of data is ready for processing by the caller.
// The values may be accessed using the Column access functions.
// Step is called again to retrieve the next row of data.
// If an error has occurred, Step returns false;
// call [Stmt.Err] or [Stmt.Reset] to get the error.
//
// https://www.sqlite.org/c3ref/step.html
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

// Err gets the last error occurred during [Stmt.Step].
// Err returns nil after [Stmt.Reset] is called.
//
// https://www.sqlite.org/c3ref/step.html
func (s *Stmt) Err() error {
	return s.err
}

// Exec is a convenience function that repeatedly calls [Stmt.Step] until it returns false,
// then calls [Stmt.Reset] to reset the statement and get any error that occurred.
func (s *Stmt) Exec() error {
	for s.Step() {
	}
	return s.Reset()
}

// BindCount gets the number of SQL parameters in a prepared statement.
//
// https://www.sqlite.org/c3ref/bind_parameter_count.html
func (s *Stmt) BindCount() int {
	r, err := s.c.api.bindCount.Call(s.c.ctx,
		uint64(s.handle))
	if err != nil {
		panic(err)
	}
	return int(r[0])
}

// BindBool binds a bool to the prepared statement.
// The leftmost SQL parameter has an index of 1.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are stored as integers 0 (false) and 1 (true).
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindBool(param int, value bool) error {
	if value {
		return s.BindInt64(param, 1)
	}
	return s.BindInt64(param, 0)
}

// BindInt binds an int to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindInt(param int, value int) error {
	return s.BindInt64(param, int64(value))
}

// BindInt64 binds an int64 to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindInt64(param int, value int64) error {
	r, err := s.c.api.bindInteger.Call(s.c.ctx,
		uint64(s.handle), uint64(param), uint64(value))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

// BindFloat binds a float64 to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindFloat(param int, value float64) error {
	r, err := s.c.api.bindFloat.Call(s.c.ctx,
		uint64(s.handle), uint64(param), math.Float64bits(value))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

// BindText binds a string to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
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

// BindBlob binds a []byte to the prepared statement.
// The leftmost SQL parameter has an index of 1.
// Binding a nil slice is the same as calling [Stmt.BindNull].
//
// https://www.sqlite.org/c3ref/bind_blob.html
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

// BindNull binds a NULL to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindNull(param int) error {
	r, err := s.c.api.bindNull.Call(s.c.ctx,
		uint64(s.handle), uint64(param))
	if err != nil {
		return err
	}
	return s.c.error(r[0])
}

// ColumnType returns the initial [Datatype] of the result column.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnType(col int) Datatype {
	r, err := s.c.api.columnType.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}
	return Datatype(r[0])
}

// ColumnBool returns the value of the result column as a bool.
// The leftmost column of the result set has the index 0.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as integers,
// with 0 converted to false and any other value to true.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnBool(col int) bool {
	if i := s.ColumnInt64(col); i != 0 {
		return true
	}
	return false
}

// ColumnInt returns the value of the result column as an int.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnInt(col int) int {
	return int(s.ColumnInt64(col))
}

// ColumnInt64 returns the value of the result column as an int64.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnInt64(col int) int64 {
	r, err := s.c.api.columnInteger.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}
	return int64(r[0])
}

// ColumnFloat returns the value of the result column as a float64.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnFloat(col int) float64 {
	r, err := s.c.api.columnFloat.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}
	return math.Float64frombits(r[0])
}

// ColumnText returns the value of the result column as a string.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
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

	mem := s.c.mem.view(ptr, uint32(r[0]))
	return string(mem)
}

// ColumnBlob appends to buf and returns
// the value of the result column as a []byte.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
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
		return buf[0:0]
	}

	r, err = s.c.api.columnBytes.Call(s.c.ctx,
		uint64(s.handle), uint64(col))
	if err != nil {
		panic(err)
	}

	mem := s.c.mem.view(ptr, uint32(r[0]))
	return append(buf[0:0], mem...)
}
