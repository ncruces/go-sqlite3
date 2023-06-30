package sqlite3

import (
	"math"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Value is any value that can be stored in a database table.
//
// https://www.sqlite.org/c3ref/value.html
type Value struct {
	c      *Conn
	handle uint32
}

// Type returns the initial [Datatype] of the value.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Type() Datatype {
	r := v.c.call(v.c.api.valueType, uint64(v.handle))
	return Datatype(r)
}

// Bool returns the value as a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as integers,
// with 0 converted to false and any other value to true.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Bool() bool {
	if i := v.Int64(); i != 0 {
		return true
	}
	return false
}

// Int returns the value as an int.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Int() int {
	return int(v.Int64())
}

// Int64 returns the value as an int64.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Int64() int64 {
	r := v.c.call(v.c.api.valueInteger, uint64(v.handle))
	return int64(r)
}

// Float returns the value as a float64.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Float() float64 {
	r := v.c.call(v.c.api.valueFloat, uint64(v.handle))
	return math.Float64frombits(r)
}

// Time returns the value as a [time.Time].
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Time(format TimeFormat) (time.Time, error) {
	var t any
	var err error
	switch v.Type() {
	case INTEGER:
		t = v.Int64()
	case FLOAT:
		t = v.Float()
	case TEXT, BLOB:
		t, err = v.Text()
		if err != nil {
			return time.Time{}, err
		}
	case NULL:
		return time.Time{}, nil
	default:
		panic(util.AssertErr())
	}
	return format.Decode(t)
}

// Text returns the value as a string.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Text() (string, error) {
	r, err := v.RawText()
	return string(r), err
}

// Blob appends to buf and returns
// the value as a []byte.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) Blob(buf []byte) ([]byte, error) {
	r, err := v.RawBlob()
	return append(buf, r...), err
}

// RawText returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) RawText() ([]byte, error) {
	r := v.c.call(v.c.api.valueText, uint64(v.handle))
	return v.rawBytes(uint32(r))
}

// RawBlob returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v *Value) RawBlob() ([]byte, error) {
	r := v.c.call(v.c.api.valueBlob, uint64(v.handle))
	return v.rawBytes(uint32(r))
}

func (v *Value) rawBytes(ptr uint32) ([]byte, error) {
	if ptr == 0 {
		r := v.c.call(v.c.api.errcode, uint64(v.c.handle))
		return nil, v.c.error(r)
	}

	r := v.c.call(v.c.api.valueBytes, uint64(v.handle))
	return util.View(v.c.mod, ptr, r), nil
}
