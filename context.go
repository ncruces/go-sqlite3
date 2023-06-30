package sqlite3

import (
	"errors"
	"math"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Context is the context in which an SQL function executes.
//
// https://www.sqlite.org/c3ref/context.html
type Context struct {
	c      *Conn
	handle uint32
}

// ResultBool sets the result of the function to a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are stored as integers 0 (false) and 1 (true).
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultBool(value bool) {
	var i int64
	if value {
		i = 1
	}
	c.ResultInt64(i)
}

// ResultInt sets the result of the function to an int.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultInt(value int) {
	c.ResultInt64(int64(value))
}

// ResultInt64 sets the result of the function to an int64.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultInt64(value int64) {
	c.c.call(c.c.api.resultInteger,
		uint64(c.handle), uint64(value))
}

// ResultFloat sets the result of the function to a float64.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultFloat(value float64) {
	c.c.call(c.c.api.resultFloat,
		uint64(c.handle), math.Float64bits(value))
}

// ResultText sets the result of the function to a string.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultText(value string) {
	ptr := c.c.newString(value)
	c.c.call(c.c.api.resultText,
		uint64(c.handle), uint64(ptr), uint64(len(value)),
		uint64(c.c.api.destructor), _UTF8)
}

// ResultBlob sets the result of the function to a []byte.
// Returning a nil slice is the same as calling [Context.ResultNull].
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultBlob(value []byte) {
	ptr := c.c.newBytes(value)
	c.c.call(c.c.api.resultBlob,
		uint64(c.handle), uint64(ptr), uint64(len(value)),
		uint64(c.c.api.destructor))
}

// BindZeroBlob sets the result of the function to a zero-filled, length n BLOB.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultZeroBlob(n int64) {
	c.c.call(c.c.api.resultZeroBlob,
		uint64(c.handle), uint64(n))
}

// ResultNull sets the result of the function to NULL.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultNull() {
	c.c.call(c.c.api.resultNull,
		uint64(c.handle))
}

// ResultTime sets the result of the function to a [time.Time].
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultTime(value time.Time, format TimeFormat) {
	if format == TimeFormatDefault {
		c.resultRFC3339Nano(value)
		return
	}
	switch v := format.Encode(value).(type) {
	case string:
		c.ResultText(v)
	case int64:
		c.ResultInt64(v)
	case float64:
		c.ResultFloat(v)
	default:
		panic(util.AssertErr())
	}
}

func (c *Context) resultRFC3339Nano(value time.Time) {
	const maxlen = uint64(len(time.RFC3339Nano))

	ptr := c.c.new(maxlen)
	buf := util.View(c.c.mod, ptr, maxlen)
	buf = value.AppendFormat(buf[:0], time.RFC3339Nano)

	c.c.call(c.c.api.resultText,
		uint64(c.handle), uint64(ptr), uint64(len(buf)),
		uint64(c.c.api.destructor), _UTF8)
}

// ResultError sets the result of the function an error.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c *Context) ResultError(err error) {
	if errors.Is(err, NOMEM) {
		c.c.call(c.c.api.resultErrorMem, uint64(c.handle))
		return
	}

	if errors.Is(err, TOOBIG) {
		c.c.call(c.c.api.resultErrorBig, uint64(c.handle))
		return
	}

	str := err.Error()
	ptr := c.c.arena.string(str)
	c.c.call(c.c.api.resultBlob,
		uint64(c.handle), uint64(ptr), uint64(len(str)))

	var code uint64
	var ecode ErrorCode
	var xcode xErrorCode
	switch {
	case errors.As(err, &xcode):
		code = uint64(xcode)
	case errors.As(err, &ecode):
		code = uint64(ecode)
	}
	if code != 0 {
		c.c.call(c.c.api.resultErrorCode,
			uint64(c.handle), uint64(xcode))
	}
}
