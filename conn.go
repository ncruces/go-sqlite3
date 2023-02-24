package sqlite3

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"

	"github.com/tetratelabs/wazero/api"
)

// Conn is a database connection handle.
//
// https://www.sqlite.org/c3ref/sqlite3.html
type Conn struct {
	ctx    context.Context
	api    sqliteAPI
	mem    memory
	handle uint32

	arena     arena
	mtx       sync.Mutex
	interrupt context.Context
	waiter    chan struct{}
	pending   *Stmt
}

// Open calls [OpenFlags] with [OPEN_READWRITE] and [OPEN_CREATE].
func Open(filename string) (conn *Conn, err error) {
	return OpenFlags(filename, OPEN_READWRITE|OPEN_CREATE)
}

// OpenFlags opens an SQLite database file as specified by the filename argument.
//
// https://www.sqlite.org/c3ref/open.html
func OpenFlags(filename string, flags OpenFlag) (conn *Conn, err error) {
	ctx := context.Background()
	module, err := sqlite3.instantiateModule(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if conn == nil {
			module.Close(ctx)
		}
	}()

	c, err := newConn(ctx, module)
	if err != nil {
		return nil, err
	}
	c.arena = c.newArena(1024)

	defer c.arena.reset()
	connPtr := c.arena.new(ptrlen)
	namePtr := c.arena.string(filename)

	r := c.call(c.api.open, uint64(namePtr), uint64(connPtr), uint64(flags), 0)

	c.handle = c.mem.readUint32(connPtr)
	if err := c.error(r[0]); err != nil {
		return nil, err
	}
	return c, nil
}

// Close closes the database connection.
//
// If the database connection is associated with unfinalized prepared statements,
// open blob handles, and/or unfinished backup objects,
// Close will leave the database connection open and return [BUSY].
//
// It is safe to close a nil, zero or closed connection.
//
// https://www.sqlite.org/c3ref/close.html
func (c *Conn) Close() error {
	if c == nil || c.handle == 0 {
		return nil
	}

	c.SetInterrupt(context.Background())

	r := c.call(c.api.close, uint64(c.handle))
	if err := c.error(r[0]); err != nil {
		return err
	}

	c.handle = 0
	return c.mem.mod.Close(c.ctx)
}

// Exec is a convenience function that allows an application to run
// multiple statements of SQL without having to use a lot of code.
//
// https://www.sqlite.org/c3ref/exec.html
func (c *Conn) Exec(sql string) error {
	c.checkInterrupt()
	defer c.arena.reset()
	sqlPtr := c.arena.string(sql)

	r := c.call(c.api.exec, uint64(c.handle), uint64(sqlPtr), 0, 0, 0)
	return c.error(r[0])
}

// Prepare calls [Conn.PrepareFlags] with no flags.
func (c *Conn) Prepare(sql string) (stmt *Stmt, tail string, err error) {
	return c.PrepareFlags(sql, 0)
}

// PrepareFlags compiles the first SQL statement in sql;
// tail is left pointing to what remains uncompiled.
// If the input text contains no SQL (if the input is an empty string or a comment),
// both stmt and err will be nil.
//
// https://www.sqlite.org/c3ref/prepare.html
func (c *Conn) PrepareFlags(sql string, flags PrepareFlag) (stmt *Stmt, tail string, err error) {
	if emptyStatement(sql) {
		return nil, "", nil
	}

	defer c.arena.reset()
	stmtPtr := c.arena.new(ptrlen)
	tailPtr := c.arena.new(ptrlen)
	sqlPtr := c.arena.string(sql)

	r := c.call(c.api.prepare, uint64(c.handle),
		uint64(sqlPtr), uint64(len(sql)+1), uint64(flags),
		uint64(stmtPtr), uint64(tailPtr))

	stmt = &Stmt{c: c}
	stmt.handle = c.mem.readUint32(stmtPtr)
	i := c.mem.readUint32(tailPtr)
	tail = sql[i-sqlPtr:]

	if err := c.error(r[0], sql); err != nil {
		return nil, "", err
	}
	if stmt.handle == 0 {
		return nil, "", nil
	}
	return
}

// GetAutocommit tests the connection for auto-commit mode.
//
// https://www.sqlite.org/c3ref/get_autocommit.html
func (c *Conn) GetAutocommit() bool {
	r := c.call(c.api.autocommit, uint64(c.handle))
	return r[0] != 0
}

// LastInsertRowID returns the rowid of the most recent successful INSERT
// on the database connection.
//
// https://www.sqlite.org/c3ref/last_insert_rowid.html
func (c *Conn) LastInsertRowID() uint64 {
	r := c.call(c.api.lastRowid, uint64(c.handle))
	return r[0]
}

// Changes returns the number of rows modified, inserted or deleted
// by the most recently completed INSERT, UPDATE or DELETE statement
// on the database connection.
//
// https://www.sqlite.org/c3ref/changes.html
func (c *Conn) Changes() uint64 {
	r := c.call(c.api.changes, uint64(c.handle))
	return r[0]
}

// SetInterrupt interrupts a long-running query when a context is done.
//
// Subsequent uses of the connection will return [INTERRUPT]
// until the context is reset by another call to SetInterrupt.
//
// For example, a timeout can be associated with a connection:
//
//	ctx, cancel := context.WithTimeout(context.TODO(), 100*time.Millisecond)
//	conn.SetInterrupt(ctx)
//	defer cancel()
//
// SetInterrupt returns the old context assigned to the connection.
//
// https://www.sqlite.org/c3ref/interrupt.html
func (c *Conn) SetInterrupt(ctx context.Context) (old context.Context) {
	// Is a waiter running?
	if c.waiter != nil {
		c.waiter <- struct{}{} // Cancel the waiter.
		<-c.waiter             // Wait for it to finish.
		c.waiter = nil
	}

	old = c.interrupt
	c.interrupt = ctx
	if ctx == nil || ctx == context.Background() || ctx == context.TODO() || ctx.Done() == nil {
		// Finalize the uncompleted SQL statement.
		if c.pending != nil {
			c.pending.Close()
			c.pending = nil
		}
		return old
	}

	// Creating an uncompleted SQL statement prevents SQLite from ignoring
	// an interrupt that comes before any other statements are started.
	if c.pending == nil {
		c.pending, _, _ = c.Prepare(`SELECT 1 UNION ALL SELECT 2`)
		c.pending.Step()
	} else {
		c.pending.Reset()
	}

	// Don't create the goroutine if we're already interrupted.
	// This happens frequently while restoring to a previously interrupted state.
	if c.checkInterrupt() {
		return old
	}

	waiter := make(chan struct{})
	c.waiter = waiter
	go func() {
		select {
		case <-waiter: // Waiter was cancelled.
			break

		case <-ctx.Done(): // Done was closed.

			c.sendInterrupt()
			// Wait for the next call to SetInterrupt.
			<-waiter
		}

		// Signal that the waiter has finished.
		waiter <- struct{}{}
	}()
	return old
}

func (c *Conn) checkInterrupt() bool {
	if c.interrupt == nil || c.interrupt.Err() == nil {
		return false
	}
	c.sendInterrupt()
	return true
}

func (c *Conn) sendInterrupt() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	// This is safe to call from a goroutine
	// because it doesn't touch the C stack.
	c.call(c.api.interrupt, uint64(c.handle))
}

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
		old := conn.SetInterrupt(context.Background())
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

func (c *Conn) error(rc uint64, sql ...string) error {
	if rc == _OK {
		return nil
	}

	err := Error{code: rc}

	if err.Code() == NOMEM || err.ExtendedCode() == IOERR_NOMEM {
		panic(oomErr)
	}

	var r []uint64

	r, _ = c.api.errstr.Call(c.ctx, rc)
	if r != nil {
		err.str = c.mem.readString(uint32(r[0]), _MAX_STRING)
	}

	r, _ = c.api.errmsg.Call(c.ctx, uint64(c.handle))
	if r != nil {
		err.msg = c.mem.readString(uint32(r[0]), _MAX_STRING)
	}

	if sql != nil {
		r, _ = c.api.erroff.Call(c.ctx, uint64(c.handle))
		if r != nil && r[0] != math.MaxUint32 {
			err.sql = sql[0][r[0]:]
		}
	}

	switch err.msg {
	case err.str, "not an error":
		err.msg = ""
	}
	return &err
}

func (c *Conn) call(fn api.Function, params ...uint64) []uint64 {
	r, err := fn.Call(c.ctx, params...)
	if err != nil {
		panic(err)
	}
	return r
}

func (c *Conn) free(ptr uint32) {
	if ptr == 0 {
		return
	}
	c.call(c.api.free, uint64(ptr))
}

func (c *Conn) new(size uint32) uint32 {
	r := c.call(c.api.malloc, uint64(size))
	ptr := uint32(r[0])
	if ptr == 0 && size != 0 {
		panic(oomErr)
	}
	return ptr
}

func (c *Conn) newBytes(b []byte) uint32 {
	if b == nil {
		return 0
	}
	ptr := c.new(uint32(len(b)))
	c.mem.writeBytes(ptr, b)
	return ptr
}

func (c *Conn) newString(s string) uint32 {
	ptr := c.new(uint32(len(s) + 1))
	c.mem.writeString(ptr, s)
	return ptr
}

func (c *Conn) newArena(size uint32) arena {
	return arena{
		c:    c,
		size: size,
		base: c.new(size),
	}
}

type arena struct {
	c    *Conn
	base uint32
	next uint32
	size uint32
	ptrs []uint32
}

func (a *arena) free() {
	if a.c == nil {
		return
	}
	a.reset()
	a.c.free(a.base)
	a.c = nil
}

func (a *arena) reset() {
	for _, ptr := range a.ptrs {
		a.c.free(ptr)
	}
	a.ptrs = nil
	a.next = 0
}

func (a *arena) new(size uint32) uint32 {
	if a.next+size <= a.size {
		ptr := a.base + a.next
		a.next += size
		return ptr
	}
	ptr := a.c.new(size)
	a.ptrs = append(a.ptrs, ptr)
	return ptr
}

func (a *arena) string(s string) uint32 {
	ptr := a.new(uint32(len(s) + 1))
	a.c.mem.writeString(ptr, s)
	return ptr
}
