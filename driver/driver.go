//go:build todo

// Package driver provides a database/sql driver for SQLite.
package driver

import (
	"database/sql"
	"database/sql/driver"

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
	return conn{c}, nil
}

type conn struct{ *sqlite3.Conn }
type stmt struct{ *sqlite3.Stmt }

func (c conn) Begin() (driver.Tx, error)

func (c conn) Prepare(query string) (driver.Stmt, error) {
	s, _, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt{s}, nil
}

func (s stmt) NumInput() int

func (s stmt) Exec(args []driver.Value) (driver.Result, error)

func (s stmt) Query(args []driver.Value) (driver.Rows, error)
