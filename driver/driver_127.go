//go:build go1.27

package driver

import (
	"database/sql"
	"math"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func (r *rows) ScanColumn(i int, dest any) error {
	typ := r.Stmt.ColumnType(i)

	var src any
	switch typ {
	case sqlite3.NULL:
		// src = nil
	case sqlite3.FLOAT:
		f := r.stmt.ColumnFloat(i)
		switch d := dest.(type) {
		case *float64:
			*d = f
			return nil
		case *float32:
			*d = float32(f)
			return nil
		case *sql.NullFloat64:
			d.Float64 = f
			d.Valid = true
			return nil
		case *sql.Null[float32]:
			d.V = float32(f)
			d.Valid = true
			return nil
		}
		src = f
	case sqlite3.INTEGER:
		i := r.stmt.ColumnInt64(i)
		switch d := dest.(type) {
		case *int64:
			*d = i
			return nil
		case *uint64:
			if 0 <= i {
				*d = uint64(i)
				return nil
			}
		case *uint:
			if 0 <= i && uint64(i) <= math.MaxUint {
				*d = uint(i)
				return nil
			}
		case *int:
			if math.MinInt <= i && i <= math.MaxInt {
				*d = int(i)
				return nil
			}
		case *sql.Null[int]:
			if math.MinInt <= i && i <= math.MaxInt {
				d.V = int(i)
				d.Valid = true
				return nil
			}
		case *sql.NullInt64:
			d.Int64 = i
			d.Valid = true
			return nil
		}
		src = i
	default:
		var b []byte
		if typ == sqlite3.TEXT {
			b = r.stmt.ColumnRawText(i)
		} else {
			b = r.stmt.ColumnRawBlob(i)
		}
		switch d := dest.(type) {
		case *sql.RawBytes:
			*d = b
			return nil
		case *string:
			*d = string(b)
			return nil
		case *[]byte:
			*d = append((*d)[:0], b...)
			return nil
		}
		src = b
	}

	// Time handling.
	var ok bool
	switch d := dest.(type) {
	case *time.Time:
		*d, ok = r.scanTime(src)
	case *sql.NullTime:
		d.Time, ok = r.scanTime(src)
		d.Valid = ok
	case *sql.Null[time.Time]:
		d.V, ok = r.scanTime(src)
		d.Valid = ok
	}
	if ok {
		return nil
	}

	// Fallback.
	return sql.ConvertAssign(dest, r.convert(i, src))
}

func (r *rows) scanTime(src any) (time.Time, bool) {
	if s, ok := src.([]byte); ok {
		if len(s) == cap(s) {
			return time.Time{}, false
		}
		if t, ok := maybeTime(s); ok {
			return t, true
		}
		src = string(s)
	}
	t, err := r.tmRead.Decode(src)
	return t, err == nil
}
