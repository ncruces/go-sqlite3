package sqlite3

import (
	"context"
	"fmt"
	"iter"
	"math"
	"math/rand"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
)

// Conn is a database connection handle.
// A Conn is not safe for concurrent use by multiple goroutines.
//
// https://sqlite.org/c3ref/sqlite3.html
type Conn struct {
	*sqlite

	interrupt  context.Context
	stmts      []*Stmt
	busy       func(context.Context, int) bool
	log        func(xErrorCode, string)
	collation  func(*Conn, string)
	wal        func(*Conn, string, int) error
	trace      func(TraceEvent, any, any) error
	authorizer func(AuthorizerActionCode, string, string, string, string) AuthorizerReturnCode
	update     func(AuthorizerActionCode, string, string, int64)
	commit     func() bool
	rollback   func()

	busy1st time.Time
	busylst time.Time
	arena   arena
	handle  ptr_t
	gosched uint8
}

// Open calls [OpenFlags] with [OPEN_READWRITE], [OPEN_CREATE] and [OPEN_URI].
func Open(filename string) (*Conn, error) {
	return newConn(context.Background(), filename, OPEN_READWRITE|OPEN_CREATE|OPEN_URI)
}

// OpenContext is like [Open] but includes a context,
// which is used to interrupt the process of opening the connection.
func OpenContext(ctx context.Context, filename string) (*Conn, error) {
	return newConn(ctx, filename, OPEN_READWRITE|OPEN_CREATE|OPEN_URI)
}

// OpenFlags opens an SQLite database file as specified by the filename argument.
//
// If none of the required flags are used, a combination of [OPEN_READWRITE] and [OPEN_CREATE] is used.
// If a URI filename is used, PRAGMA statements to execute can be specified using "_pragma":
//
//	sqlite3.Open("file:demo.db?_pragma=busy_timeout(10000)")
//
// https://sqlite.org/c3ref/open.html
func OpenFlags(filename string, flags OpenFlag) (*Conn, error) {
	if flags&(OPEN_READONLY|OPEN_READWRITE|OPEN_CREATE) == 0 {
		flags |= OPEN_READWRITE | OPEN_CREATE
	}
	return newConn(context.Background(), filename, flags)
}

type connKey = util.ConnKey

func newConn(ctx context.Context, filename string, flags OpenFlag) (ret *Conn, _ error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	c := &Conn{interrupt: ctx}
	c.sqlite, err = instantiateSQLite()
	if err != nil {
		return nil, err
	}
	defer func() {
		if ret == nil {
			c.Close()
			c.sqlite.close()
		} else {
			c.interrupt = context.Background()
		}
	}()

	c.ctx = context.WithValue(c.ctx, connKey{}, c)
	if logger := defaultLogger.Load(); logger != nil {
		c.ConfigLog(*logger)
	}
	c.arena = c.newArena()
	c.handle, err = c.openDB(filename, flags)
	if err == nil {
		err = initExtensions(c)
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) openDB(filename string, flags OpenFlag) (ptr_t, error) {
	defer c.arena.mark()()
	connPtr := c.arena.new(ptrlen)
	namePtr := c.arena.string(filename)

	flags |= OPEN_EXRESCODE
	rc := res_t(c.mod.Xsqlite3_open_v2(int32(namePtr), int32(connPtr), int32(flags), 0))

	handle := util.Read32[ptr_t](c.mod, connPtr)
	if err := c.sqlite.error(rc, handle); err != nil {
		c.closeDB(handle)
		return 0, err
	}

	c.mod.Xsqlite3_progress_handler_go(int32(handle), 1000)
	if flags|OPEN_URI != 0 && strings.HasPrefix(filename, "file:") {
		var pragmas strings.Builder
		if _, after, ok := strings.Cut(filename, "?"); ok {
			query, _ := url.ParseQuery(after)
			for _, p := range query["_pragma"] {
				pragmas.WriteString(`PRAGMA `)
				pragmas.WriteString(p)
				pragmas.WriteString(`;`)
			}
		}
		if pragmas.Len() != 0 {
			pragmaPtr := c.arena.string(pragmas.String())
			rc := res_t(c.mod.Xsqlite3_exec(int32(handle), int32(pragmaPtr), 0, 0, 0))
			if err := c.sqlite.error(rc, handle, pragmas.String()); err != nil {
				err = fmt.Errorf("sqlite3: invalid _pragma: %w", err)
				c.closeDB(handle)
				return 0, err
			}
		}
	}
	return handle, nil
}

func (c *Conn) closeDB(handle ptr_t) {
	rc := res_t(c.mod.Xsqlite3_close_v2(int32(handle)))
	if err := c.sqlite.error(rc, handle); err != nil {
		panic(err)
	}
}

// Close closes the database connection.
//
// If the database connection is associated with unfinalized prepared statements,
// open blob handles, and/or unfinished backup objects,
// Close will leave the database connection open and return [BUSY].
//
// It is safe to close a nil, zero or closed Conn.
//
// https://sqlite.org/c3ref/close.html
func (c *Conn) Close() error {
	if c == nil || c.handle == 0 {
		return nil
	}

	rc := res_t(c.mod.Xsqlite3_close(int32(c.handle)))
	if err := c.error(rc); err != nil {
		return err
	}

	c.handle = 0
	return c.close()
}

// Exec is a convenience function that allows an application to run
// multiple statements of SQL without having to use a lot of code.
//
// https://sqlite.org/c3ref/exec.html
func (c *Conn) Exec(sql string) error {
	if c.interrupt.Err() != nil {
		return INTERRUPT
	}
	return c.exec(sql)
}

func (c *Conn) exec(sql string) error {
	defer c.arena.mark()()
	textPtr := c.arena.string(sql)
	rc := res_t(c.mod.Xsqlite3_exec(int32(c.handle), int32(textPtr), 0, 0, 0))
	return c.error(rc, sql)
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
// https://sqlite.org/c3ref/prepare.html
func (c *Conn) PrepareFlags(sql string, flags PrepareFlag) (stmt *Stmt, tail string, err error) {
	if len(sql) > _MAX_SQL_LENGTH {
		return nil, "", TOOBIG
	}
	if c.interrupt.Err() != nil {
		return nil, "", INTERRUPT
	}

	defer c.arena.mark()()
	stmtPtr := c.arena.new(ptrlen)
	tailPtr := c.arena.new(ptrlen)
	textPtr := c.arena.string(sql)

	rc := res_t(c.mod.Xsqlite3_prepare_v3(int32(c.handle),
		int32(textPtr), int32(len(sql)+1), int32(flags),
		int32(stmtPtr), int32(tailPtr)))

	stmt = &Stmt{c: c, sql: sql}
	stmt.handle = util.Read32[ptr_t](c.mod, stmtPtr)
	if sql := sql[util.Read32[ptr_t](c.mod, tailPtr)-textPtr:]; sql != "" {
		tail = sql
	}

	if err := c.error(rc, sql); err != nil {
		return nil, "", err
	}
	if stmt.handle == 0 {
		return nil, "", nil
	}
	c.stmts = append(c.stmts, stmt)
	return stmt, tail, nil
}

// DBName returns the schema name for n-th database on the database connection.
//
// https://sqlite.org/c3ref/db_name.html
func (c *Conn) DBName(n int) string {
	ptr := ptr_t(c.mod.Xsqlite3_db_name(int32(c.handle), int32(n)))
	if ptr == 0 {
		return ""
	}
	return util.ReadString(c.mod, ptr, _MAX_NAME)
}

// Filename returns the filename for a database.
//
// https://sqlite.org/c3ref/db_filename.html
func (c *Conn) Filename(schema string) *vfs.Filename {
	var ptr ptr_t
	if schema != "" {
		defer c.arena.mark()()
		ptr = c.arena.string(schema)
	}
	ptr = ptr_t(c.mod.Xsqlite3_db_filename(int32(c.handle), int32(ptr)))
	return vfs.GetFilename(c.ctx, c.mod, ptr, vfs.OPEN_MAIN_DB)
}

// ReadOnly determines if a database is read-only.
//
// https://sqlite.org/c3ref/db_readonly.html
func (c *Conn) ReadOnly(schema string) (ro bool, ok bool) {
	var ptr ptr_t
	if schema != "" {
		defer c.arena.mark()()
		ptr = c.arena.string(schema)
	}
	b := c.mod.Xsqlite3_db_readonly(int32(c.handle), int32(ptr))
	return b > 0, b < 0
}

// GetAutocommit tests the connection for auto-commit mode.
//
// https://sqlite.org/c3ref/get_autocommit.html
func (c *Conn) GetAutocommit() bool {
	b := c.mod.Xsqlite3_get_autocommit(int32(c.handle))
	return b != 0
}

// LastInsertRowID returns the rowid of the most recent successful INSERT
// on the database connection.
//
// https://sqlite.org/c3ref/last_insert_rowid.html
func (c *Conn) LastInsertRowID() int64 {
	return c.mod.Xsqlite3_last_insert_rowid(int32(c.handle))
}

// SetLastInsertRowID allows the application to set the value returned by
// [Conn.LastInsertRowID].
//
// https://sqlite.org/c3ref/set_last_insert_rowid.html
func (c *Conn) SetLastInsertRowID(id int64) {
	c.mod.Xsqlite3_set_last_insert_rowid(int32(c.handle), id)
}

// Changes returns the number of rows modified, inserted or deleted
// by the most recently completed INSERT, UPDATE or DELETE statement
// on the database connection.
//
// https://sqlite.org/c3ref/changes.html
func (c *Conn) Changes() int64 {
	return c.mod.Xsqlite3_changes64(int32(c.handle))
}

// TotalChanges returns the number of rows modified, inserted or deleted
// by all INSERT, UPDATE or DELETE statements completed
// since the database connection was opened.
//
// https://sqlite.org/c3ref/total_changes.html
func (c *Conn) TotalChanges() int64 {
	return c.mod.Xsqlite3_total_changes64(int32(c.handle))
}

// ReleaseMemory frees memory used by a database connection.
//
// https://sqlite.org/c3ref/db_release_memory.html
func (c *Conn) ReleaseMemory() error {
	rc := res_t(c.mod.Xsqlite3_db_release_memory(int32(c.handle)))
	return c.error(rc)
}

// GetInterrupt gets the context set with [Conn.SetInterrupt].
func (c *Conn) GetInterrupt() context.Context {
	return c.interrupt
}

// SetInterrupt interrupts a long-running query when a context is done.
//
// Subsequent uses of the connection will return [INTERRUPT]
// until the context is reset by another call to SetInterrupt.
//
// To associate a timeout with a connection:
//
//	ctx, cancel := context.WithTimeout(context.TODO(), 100*time.Millisecond)
//	conn.SetInterrupt(ctx)
//	defer cancel()
//
// SetInterrupt returns the old context assigned to the connection.
//
// https://sqlite.org/c3ref/interrupt.html
func (c *Conn) SetInterrupt(ctx context.Context) (old context.Context) {
	if ctx == nil {
		panic("nil Context")
	}
	old = c.interrupt
	c.interrupt = ctx
	return old
}

func (sqlt *sqlite) Xgo_progress_handler(_ int32) (interrupt int32) {
	if c, ok := sqlt.ctx.Value(connKey{}).(*Conn); ok {
		if c.gosched++; c.gosched%16 == 0 {
			runtime.Gosched()
		}
		if c.interrupt.Err() != nil {
			interrupt = 1
		}
	}
	return interrupt
}

// BusyTimeout sets a busy timeout.
//
// https://sqlite.org/c3ref/busy_timeout.html
func (c *Conn) BusyTimeout(timeout time.Duration) error {
	ms := min((timeout+time.Millisecond-1)/time.Millisecond, math.MaxInt32)
	rc := res_t(c.mod.Xsqlite3_busy_timeout(int32(c.handle), int32(ms)))
	return c.error(rc)
}

func (sqlt *sqlite) Xgo_busy_timeout(count, tmout int32) (retry int32) {
	// https://fractaledmind.github.io/2024/04/15/sqlite-on-rails-the-how-and-why-of-optimal-performance/
	if c, ok := sqlt.ctx.Value(connKey{}).(*Conn); ok && c.interrupt.Err() == nil {
		switch {
		case count == 0:
			c.busy1st = time.Now()
		case time.Since(c.busy1st) >= time.Duration(tmout)*time.Millisecond:
			return 0
		}
		if time.Since(c.busylst) < time.Millisecond {
			const sleepIncrement = 2*1024*1024 - 1 // power of two, ~2ms
			time.Sleep(time.Duration(rand.Int63() & sleepIncrement))
		}
		c.busylst = time.Now()
		return 1
	}
	return 0
}

// BusyHandler registers a callback to handle [BUSY] errors.
//
// https://sqlite.org/c3ref/busy_handler.html
func (c *Conn) BusyHandler(cb func(ctx context.Context, count int) (retry bool)) error {
	var enable int32
	if cb != nil {
		enable = 1
	}
	rc := res_t(c.mod.Xsqlite3_busy_handler_go(int32(c.handle), enable))
	if err := c.error(rc); err != nil {
		return err
	}
	c.busy = cb
	return nil
}

func (sqlt *sqlite) Xgo_busy_handler(pDB, count int32) (retry int32) {
	if c, ok := sqlt.ctx.Value(connKey{}).(*Conn); ok && c.handle == ptr_t(pDB) && c.busy != nil {
		if interrupt := c.interrupt; interrupt.Err() == nil &&
			c.busy(interrupt, int(count)) {
			retry = 1
		}
	}
	return retry
}

// Status retrieves runtime status information about a database connection.
//
// https://sqlite.org/c3ref/db_status.html
func (c *Conn) Status(op DBStatus, reset bool) (current, highwater int64, err error) {
	defer c.arena.mark()()
	hiPtr := c.arena.new(8)
	curPtr := c.arena.new(8)

	var i int32
	if reset {
		i = 1
	}

	rc := res_t(c.mod.Xsqlite3_db_status64(int32(c.handle),
		int32(op), int32(curPtr), int32(hiPtr), i))
	if err = c.error(rc); err == nil {
		current = util.Read64[int64](c.mod, curPtr)
		highwater = util.Read64[int64](c.mod, hiPtr)
	}
	return
}

// TableColumnMetadata extracts metadata about a column of a table.
//
// https://sqlite.org/c3ref/table_column_metadata.html
func (c *Conn) TableColumnMetadata(schema, table, column string) (declType, collSeq string, notNull, primaryKey, autoInc bool, err error) {
	defer c.arena.mark()()
	var (
		declTypePtr   ptr_t
		collSeqPtr    ptr_t
		notNullPtr    ptr_t
		primaryKeyPtr ptr_t
		autoIncPtr    ptr_t
		columnPtr     ptr_t
		schemaPtr     ptr_t
	)
	if column != "" {
		declTypePtr = c.arena.new(ptrlen)
		collSeqPtr = c.arena.new(ptrlen)
		notNullPtr = c.arena.new(ptrlen)
		primaryKeyPtr = c.arena.new(ptrlen)
		autoIncPtr = c.arena.new(ptrlen)
		columnPtr = c.arena.string(column)
	}
	if schema != "" {
		schemaPtr = c.arena.string(schema)
	}
	tablePtr := c.arena.string(table)

	rc := res_t(c.mod.Xsqlite3_table_column_metadata(int32(c.handle),
		int32(schemaPtr), int32(tablePtr), int32(columnPtr),
		int32(declTypePtr), int32(collSeqPtr),
		int32(notNullPtr), int32(primaryKeyPtr), int32(autoIncPtr)))
	if err = c.error(rc); err == nil && column != "" {
		if ptr := util.Read32[ptr_t](c.mod, declTypePtr); ptr != 0 {
			declType = util.ReadString(c.mod, ptr, _MAX_NAME)
		}
		if ptr := util.Read32[ptr_t](c.mod, collSeqPtr); ptr != 0 {
			collSeq = util.ReadString(c.mod, ptr, _MAX_NAME)
		}
		notNull = util.ReadBool(c.mod, notNullPtr)
		autoInc = util.ReadBool(c.mod, autoIncPtr)
		primaryKey = util.ReadBool(c.mod, primaryKeyPtr)
	}
	return
}

func (c *Conn) error(rc res_t, sql ...string) error {
	return c.sqlite.error(rc, c.handle, sql...)
}

// Stmts returns an iterator for the prepared statements
// associated with the database connection.
//
// https://sqlite.org/c3ref/next_stmt.html
func (c *Conn) Stmts() iter.Seq[*Stmt] {
	return func(yield func(*Stmt) bool) {
		for _, s := range c.stmts {
			if !yield(s) {
				break
			}
		}
	}
}
