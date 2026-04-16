//go:build go1.27

package driver

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func (r *rows) ScanColumn(i int, dest any) error {
	typ := r.Stmt.ColumnType(i)

	// Fast path.
	switch d := dest.(type) {
	case *int:
		if strconv.IntSize == 64 && typ == sqlite3.INTEGER {
			*d = r.Stmt.ColumnInt(i)
			return nil
		}
	case *int64:
		if typ == sqlite3.INTEGER {
			*d = r.Stmt.ColumnInt64(i)
			return nil
		}
	case *float64:
		if typ == sqlite3.FLOAT {
			*d = r.Stmt.ColumnFloat(i)
			return nil
		}
	case *string:
		if typ == sqlite3.BLOB || typ == sqlite3.TEXT {
			*d = r.stmt.ColumnText(i)
			return nil
		}
	case *[]byte:
		if typ == sqlite3.BLOB || typ == sqlite3.TEXT {
			*d = r.stmt.ColumnBlob(i, (*d)[:0])
			return nil
		}
	case *sql.RawBytes:
		if typ == sqlite3.BLOB || typ == sqlite3.TEXT {
			*d = r.stmt.ColumnRawBlob(i)
			return nil
		}
	}

	var src any
	switch typ {
	case sqlite3.NULL:
		//
	case sqlite3.INTEGER:
		src = r.stmt.ColumnInt64(i)
	case sqlite3.FLOAT:
		src = r.stmt.ColumnFloat(i)
	case sqlite3.TEXT:
		src = r.stmt.ColumnRawText(i)
	case sqlite3.BLOB:
		src = r.stmt.ColumnRawBlob(i)
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
	if s, ok := src.([]byte); ok && len(s) != cap(s) {
		src = string(s)
	}
	t, err := r.tmRead.Decode(src)
	return t, err == nil
}
