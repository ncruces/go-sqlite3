// Package driver provides a database/sql driver for SQLite.
package driver

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func init() {
	sql.Register("sqlite3", sqlite{})
}

type sqlite struct{}

func (sqlite) Open(name string) (driver.Conn, error) {
	c, err := sqlite3.OpenFlags(name, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE|sqlite3.OPEN_URI)
	if err != nil {
		return nil, err
	}
	// If the database is not in WAL mode,
	// use normal locking mode.
	journal, err := pragma(c, "journal_mode")
	if err != nil {
		return nil, err
	}
	if journal != "wal" {
		pragma(c, "locking_mode=normal")
	}
	return conn{c}, nil
}

type conn struct{ conn *sqlite3.Conn }

var (
	// Ensure these interfaces are implemented:
	_ driver.Validator = conn{}
	// _ driver.SessionResetter = conn{}
	// _ driver.ExecerContext   = conn{}
	// _ driver.QueryerContext  = conn{}
	// _ driver.ConnBeginTx     = conn{}
)

func (c conn) Close() error {
	return c.conn.Close()
}

func (c conn) IsValid() bool {
	// Pool only normal locking mode connections.
	mode, _ := pragma(c.conn, "locking_mode")
	return mode == "normal"
}

func (c conn) Begin() (driver.Tx, error) {
	err := c.conn.Exec(`BEGIN`)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c conn) Commit() error {
	err := c.conn.Exec(`COMMIT`)
	if err != nil {
		c.Rollback()
	}
	return err
}

func (c conn) Rollback() error {
	return c.conn.Exec(`ROLLBACK`)
}

func (c conn) Prepare(query string) (driver.Stmt, error) {
	s, _, err := c.conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt{s, c.conn}, nil
}

func pragma(c *sqlite3.Conn, pragma string) (string, error) {
	stmt, _, err := c.Prepare(`PRAGMA ` + pragma)
	if err != nil {
		return "", err
	}
	defer stmt.Close()
	if stmt.Step() {
		return stmt.ColumnText(0), nil
	}
	return "", stmt.Err()
}

type stmt struct {
	stmt *sqlite3.Stmt
	conn *sqlite3.Conn
}

var (
	// Ensure these interfaces are implemented:
	// _ driver.StmtExecContext  = stmt{}
	// _ driver.StmtQueryContext = stmt{}
	_ = stmt{}
)

func (s stmt) Close() error {
	return s.stmt.Close()
}

func (s stmt) NumInput() int {
	return s.stmt.BindCount()
}

func (s stmt) Exec(args []driver.Value) (driver.Result, error) {
	_, err := s.Query(args)
	if err != nil {
		return nil, err
	}

	err = s.stmt.Exec()
	if err != nil {
		return nil, err
	}

	return result{
		int64(s.conn.LastInsertRowID()),
		int64(s.conn.Changes()),
	}, nil
}

func (s stmt) Query(args []driver.Value) (driver.Rows, error) {
	var err error
	for i, arg := range args {
		switch a := arg.(type) {
		case bool:
			err = s.stmt.BindBool(i+1, a)
		case int64:
			err = s.stmt.BindInt64(i+1, a)
		case float64:
			err = s.stmt.BindFloat(i+1, a)
		case string:
			err = s.stmt.BindText(i+1, a)
		case []byte:
			err = s.stmt.BindBlob(i+1, a)
		case time.Time:
			err = s.stmt.BindText(i+1, a.Format(time.RFC3339Nano))
		case nil:
			err = s.stmt.BindNull(i + 1)
		default:
			panic(assertErr)
		}
		if err != nil {
			return nil, err
		}
	}
	return rows{s.stmt}, nil
}

type result struct{ lastInsertId, rowsAffected int64 }

func (r result) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

type rows struct{ s *sqlite3.Stmt }

func (r rows) Close() error {
	return r.s.Reset()
}

func (r rows) Columns() []string {
	count := r.s.ColumnCount()
	columns := make([]string, count)
	for i := range columns {
		columns[i] = r.s.ColumnName(i)
	}
	return columns
}

func (r rows) Next(dest []driver.Value) error {
	if !r.s.Step() {
		if err := r.s.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	for i := range dest {
		switch r.s.ColumnType(i) {
		case sqlite3.INTEGER:
			dest[i] = r.s.ColumnInt64(i)
		case sqlite3.FLOAT:
			dest[i] = r.s.ColumnFloat(i)
		case sqlite3.TEXT:
			dest[i] = maybeDate(r.s.ColumnText(i))
		case sqlite3.BLOB:
			buf, _ := dest[i].([]byte)
			dest[i] = r.s.ColumnBlob(i, buf)
		case sqlite3.NULL:
			if buf, ok := dest[i].([]byte); ok {
				dest[i] = buf[0:0]
			} else {
				dest[i] = nil
			}
		default:
			panic(assertErr)
		}
	}

	return r.s.Err()
}
