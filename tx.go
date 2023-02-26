package sqlite3

import (
	"context"
	"errors"
	"fmt"
	"runtime"
)

type Tx struct {
	c *Conn
}

// Begin starts a deferred transaction.
//
// https://www.sqlite.org/lang_transaction.html
func (c *Conn) Begin() Tx {
	err := c.Exec(`BEGIN DEFERRED`)
	if err != nil && !errors.Is(err, INTERRUPT) {
		panic(err)
	}
	return Tx{c}
}

// BeginImmediate starts an immediate transaction.
//
// https://www.sqlite.org/lang_transaction.html
func (c *Conn) BeginImmediate() (Tx, error) {
	err := c.Exec(`BEGIN IMMEDIATE`)
	if err != nil {
		return Tx{}, err
	}
	return Tx{c}, nil
}

// BeginExclusive starts an exclusive transaction.
//
// https://www.sqlite.org/lang_transaction.html
func (c *Conn) BeginExclusive() (Tx, error) {
	err := c.Exec(`BEGIN EXCLUSIVE`)
	if err != nil {
		return Tx{}, err
	}
	return Tx{c}, nil
}

// End calls either [Commit] or [Rollback]
// depending on whether *error points to a nil or non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(conn *sqlite3.Conn) (err error) {
//		tx := conn.Begin()
//		defer tx.End(&err)
//
//		// ... do work in the transaction
//	}
//
// https://www.sqlite.org/lang_savepoint.html
func (tx Tx) End(errp *error) {
	recovered := recover()
	if recovered != nil {
		defer panic(recovered)
	}

	if tx.c.GetAutocommit() {
		// There is nothing to commit/rollback.
		return
	}

	if *errp == nil && recovered == nil {
		// Success path.
		*errp = tx.Commit()
		if *errp == nil {
			return
		}
		// Possible interrupt, fall through to the error path.
	}

	// Error path.
	err := tx.Rollback()
	if err != nil {
		panic(err)
	}
}

func (tx Tx) Commit() error {
	return tx.c.Exec(`COMMIT`)
}

func (tx Tx) Rollback() error {
	// ROLLBACK even if the connection has been interrupted.
	old := tx.c.SetInterrupt(context.Background())
	defer tx.c.SetInterrupt(old)
	return tx.c.Exec(`ROLLBACK`)
}

// Savepoint creates a named SQLite transaction using SAVEPOINT.
//
// On success Savepoint returns a release func that will call either
// RELEASE or ROLLBACK depending on whether the parameter *error
// points to a nil or non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(conn *sqlite3.Conn) (err error) {
//		defer conn.Savepoint()(&err)
//
//		// ... do work in the transaction
//	}
//
// https://www.sqlite.org/lang_savepoint.html
func (c *Conn) Savepoint() (release func(*error)) {
	name := "sqlite3.Savepoint" // names can be reused
	var pc [1]uintptr
	if n := runtime.Callers(2, pc[:]); n > 0 {
		frames := runtime.CallersFrames(pc[:n])
		frame, _ := frames.Next()
		if frame.Function != "" {
			name = frame.Function
		}
	}

	err := c.Exec(fmt.Sprintf("SAVEPOINT %q;", name))
	if err != nil {
		if errors.Is(err, INTERRUPT) {
			return func(errp *error) {
				if *errp == nil {
					*errp = err
				}
			}
		}
		panic(err)
	}

	return func(errp *error) {
		recovered := recover()
		if recovered != nil {
			defer panic(recovered)
		}

		if c.GetAutocommit() {
			// There is nothing to commit/rollback.
			return
		}

		if *errp == nil && recovered == nil {
			// Success path.
			// RELEASE the savepoint successfully.
			*errp = c.Exec(fmt.Sprintf("RELEASE %q;", name))
			if *errp == nil {
				return
			}
			// Possible interrupt, fall through to the error path.
		}

		// Error path.
		// Always ROLLBACK even if the connection has been interrupted.
		old := c.SetInterrupt(context.Background())
		defer c.SetInterrupt(old)

		err := c.Exec(fmt.Sprintf("ROLLBACK TO %q;", name))
		if err != nil {
			panic(err)
		}
		err = c.Exec(fmt.Sprintf("RELEASE %q;", name))
		if err != nil {
			panic(err)
		}
	}
}
