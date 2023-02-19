// Package driver provides a database/sql driver for SQLite.
package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func init() {
	sql.Register("sqlite3", sqlite{})
}

type sqlite struct{}

func (sqlite) Open(name string) (driver.Conn, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	c, err := sqlite3.OpenFlags(name, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE|sqlite3.OPEN_URI|sqlite3.OPEN_EXRESCODE)
	if err != nil {
		return nil, err
	}
	var pragmas strings.Builder
	for _, p := range u.Query()["_pragma"] {
		pragmas.WriteString(`PRAGMA `)
		pragmas.WriteString(p)
		pragmas.WriteByte(';')
	}
	if pragmas.Len() == 0 {
		pragmas.WriteString(`PRAGMA locking_mode=normal;`)
		pragmas.WriteString(`PRAGMA busy_timeout=60000;`)
	}
	err = c.Exec(pragmas.String())
	if err != nil {
		return nil, err
	}
	return conn{c, pragmas.String()}, nil
}

type conn struct {
	conn    *sqlite3.Conn
	pragmas string
}

var (
	// Ensure these interfaces are implemented:
	_ driver.Validator       = conn{}
	_ driver.SessionResetter = conn{}
	_ driver.ExecerContext   = conn{}
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

func (c conn) ResetSession(ctx context.Context) error {
	return c.conn.Exec(c.pragmas)
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
	s, tail, err := c.conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	if tail != "" {
		// Check if the tail contains any SQL.
		st, _, err := c.conn.Prepare(tail)
		if err != nil {
			s.Close()
			return nil, err
		}
		if st != nil {
			s.Close()
			st.Close()
			return nil, tailErr
		}
	}
	return stmt{s, c.conn}, nil
}

func (c conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 0 {
		// Slow path.
		return nil, driver.ErrSkip
	}

	ch := c.conn.SetInterrupt(ctx.Done())
	defer c.conn.SetInterrupt(ch)

	err := c.conn.Exec(query)
	if err != nil {
		return nil, err
	}

	return result{
		int64(c.conn.LastInsertRowID()),
		int64(c.conn.Changes()),
	}, nil
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
	_ driver.StmtExecContext  = stmt{}
	_ driver.StmtQueryContext = stmt{}
)

func (s stmt) Close() error {
	return s.stmt.Close()
}

func (s stmt) NumInput() int {
	return s.stmt.BindCount()
}

// Deprecated: use ExecContext instead.
func (s stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), namedValues(args))
}

// Deprecated: use QueryContext instead.
func (s stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), namedValues(args))
}

func (s stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	_, err := s.QueryContext(ctx, args)
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

func (s stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	err := s.stmt.ClearBindings()
	if err != nil {
		return nil, err
	}

	var ids [3]int
	for _, arg := range args {
		ids := ids[:0]
		if arg.Name == "" {
			ids = append(ids, arg.Ordinal)
		} else {
			for _, prefix := range []string{":", "@", "$"} {
				if id := s.stmt.BindIndex(prefix + arg.Name); id != 0 {
					ids = append(ids, id)
				}
			}
		}

		for _, id := range ids {
			switch a := arg.Value.(type) {
			case bool:
				err = s.stmt.BindBool(id, a)
			case int64:
				err = s.stmt.BindInt64(id, a)
			case float64:
				err = s.stmt.BindFloat(id, a)
			case string:
				err = s.stmt.BindText(id, a)
			case []byte:
				err = s.stmt.BindBlob(id, a)
			case time.Time:
				err = s.stmt.BindText(id, a.Format(time.RFC3339Nano))
			case nil:
				err = s.stmt.BindNull(id)
			default:
				panic(assertErr)
			}
		}
		if err != nil {
			return nil, err
		}
	}

	return rows{ctx, s.stmt, s.conn}, nil
}

type result struct{ lastInsertId, rowsAffected int64 }

func (r result) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

type rows struct {
	ctx  context.Context
	stmt *sqlite3.Stmt
	conn *sqlite3.Conn
}

func (r rows) Close() error {
	return r.stmt.Reset()
}

func (r rows) Columns() []string {
	count := r.stmt.ColumnCount()
	columns := make([]string, count)
	for i := range columns {
		columns[i] = r.stmt.ColumnName(i)
	}
	return columns
}

func (r rows) Next(dest []driver.Value) error {
	ch := r.conn.SetInterrupt(r.ctx.Done())
	defer r.conn.SetInterrupt(ch)

	if !r.stmt.Step() {
		if err := r.stmt.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	for i := range dest {
		switch r.stmt.ColumnType(i) {
		case sqlite3.INTEGER:
			dest[i] = r.stmt.ColumnInt64(i)
		case sqlite3.FLOAT:
			dest[i] = r.stmt.ColumnFloat(i)
		case sqlite3.TEXT:
			dest[i] = maybeDate(r.stmt.ColumnText(i))
		case sqlite3.BLOB:
			buf, _ := dest[i].([]byte)
			dest[i] = r.stmt.ColumnBlob(i, buf)
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

	return r.stmt.Err()
}
