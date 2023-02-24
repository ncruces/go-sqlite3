package sqlite3

import (
	"fmt"
	"runtime"
)

// Savepoint creates a named SQLite transaction using SAVEPOINT.
//
// On success Savepoint returns a release func that will call
// either RELEASE or ROLLBACK depending on whether the parameter *error
// points to a nil or non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(conn *sqlite3.Conn) (err error) {
//		defer conn.Savepoint()(&err)
//
//		// ... do work in the transaction
//	}
func (conn *Conn) Savepoint() (release func(*error)) {
	name := "sqlite3.Savepoint" // names can be reused
	var pc [1]uintptr
	if n := runtime.Callers(2, pc[:]); n > 0 {
		frames := runtime.CallersFrames(pc[:n])
		frame, _ := frames.Next()
		if frame.Function != "" {
			name = frame.Function
		}
	}

	err := conn.Exec(fmt.Sprintf("SAVEPOINT %q;", name))
	if err != nil {
		return func(errp *error) {
			if *errp == nil {
				*errp = err
			}
		}
	}

	return func(errp *error) {
		recovered := recover()
		if recovered != nil {
			defer panic(recovered)
		}

		if conn.GetAutocommit() {
			// There is nothing to commit/rollback.
			return
		}

		if *errp == nil && recovered == nil {
			// Success path.
			// RELEASE the savepoint successfully.
			*errp = conn.Exec(fmt.Sprintf("RELEASE %q;", name))
			if *errp == nil {
				return
			}
			// Possible interrupt, fall through to the error path.
		}

		// Error path.
		// Always ROLLBACK even if the connection has been interrupted.
		old := conn.SetInterrupt(nil)
		defer conn.SetInterrupt(old)

		err := conn.Exec(fmt.Sprintf("ROLLBACK TO %q;", name))
		if err != nil {
			panic(err)
		}
		err = conn.Exec(fmt.Sprintf("RELEASE %q;", name))
		if err != nil {
			panic(err)
		}
	}
}
