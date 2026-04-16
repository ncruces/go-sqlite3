//go:build go1.27

package driver

import (
	"database/sql"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func (r *rows) ScanColumn(i int, dest any) error {
	var tm *time.Time
	switch d := dest.(type) {
	case *time.Time:
		tm = d
	case *sql.NullTime:
		tm = &d.Time
	case *sql.Null[time.Time]:
		tm = &d.V
	}
	if tm != nil {
		t := r.Stmt.ColumnTime(i, r.tmRead)
		if !t.IsZero() {
			*tm = t
			return nil
		}
	}

	var src any
	switch r.stmt.ColumnType(i) {
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
	return sql.ConvertAssign(dest, r.convert(i, src))
}
