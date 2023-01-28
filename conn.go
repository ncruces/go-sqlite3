package sqlite3

import (
	"context"
)

type Conn struct {
	ctx    context.Context
	api    sqliteAPI
	mem    memory
	handle uint32
}

func Open(filename string) (conn *Conn, err error) {
	return OpenFlags(filename, OPEN_READWRITE|OPEN_CREATE)
}

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

	c := newConn(module)
	c.ctx = context.Background()
	namePtr := c.newString(filename)
	connPtr := c.new(ptrlen)
	defer c.free(namePtr)
	defer c.free(connPtr)

	r, err := c.api.open.Call(c.ctx, uint64(namePtr), uint64(connPtr), uint64(flags), 0)
	if err != nil {
		return nil, err
	}

	c.handle = c.mem.readUint32(connPtr)
	if err := c.error(r[0]); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) Close() error {
	r, err := c.api.close.Call(c.ctx, uint64(c.handle))
	if err != nil {
		return err
	}

	if err := c.error(r[0]); err != nil {
		return err
	}
	return c.mem.mod.Close(c.ctx)
}

func (c *Conn) Exec(sql string) error {
	sqlPtr := c.newString(sql)
	defer c.free(sqlPtr)

	r, err := c.api.exec.Call(c.ctx, uint64(c.handle), uint64(sqlPtr), 0, 0, 0)
	if err != nil {
		return err
	}
	return c.error(r[0])
}

func (c *Conn) Prepare(sql string) (stmt *Stmt, tail string, err error) {
	return c.PrepareFlags(sql, 0)
}

func (c *Conn) PrepareFlags(sql string, flags PrepareFlag) (stmt *Stmt, tail string, err error) {
	sqlPtr := c.newString(sql)
	stmtPtr := c.new(ptrlen)
	tailPtr := c.new(ptrlen)
	defer c.free(sqlPtr)
	defer c.free(stmtPtr)
	defer c.free(tailPtr)

	r, err := c.api.prepare.Call(c.ctx, uint64(c.handle),
		uint64(sqlPtr), uint64(len(sql)+1), uint64(flags),
		uint64(stmtPtr), uint64(tailPtr))
	if err != nil {
		return nil, "", err
	}

	stmt = &Stmt{c: c}
	stmt.handle = c.mem.readUint32(stmtPtr)
	i := c.mem.readUint32(tailPtr)
	tail = sql[i-sqlPtr:]

	if err := c.error(r[0]); err != nil {
		return nil, "", err
	}
	if stmt.handle == 0 {
		return nil, "", nil
	}
	return
}

func (c *Conn) error(rc uint64) error {
	if rc == _OK {
		return nil
	}

	err := Error{
		Code:         ErrorCode(rc),
		ExtendedCode: ExtendedErrorCode(rc),
	}

	if err.Code == NOMEM || err.ExtendedCode == IOERR_NOMEM {
		panic(oomErr)
	}

	var r []uint64

	// Do this first, sqlite3_errmsg is guaranteed to never change the value of the error code.
	r, _ = c.api.errmsg.Call(c.ctx, uint64(c.handle))
	if r != nil {
		err.msg = c.mem.readString(uint32(r[0]), 512)
	}

	r, _ = c.api.errstr.Call(c.ctx, rc)
	if r != nil {
		err.str = c.mem.readString(uint32(r[0]), 512)
	}

	if err.msg == err.str {
		err.msg = ""

	}
	return &err
}

func (c *Conn) free(ptr uint32) {
	if ptr == 0 {
		return
	}
	_, err := c.api.free.Call(c.ctx, uint64(ptr))
	if err != nil {
		panic(err)
	}
}

func (c *Conn) new(len uint32) uint32 {
	r, err := c.api.malloc.Call(c.ctx, uint64(len))
	if err != nil {
		panic(err)
	}
	ptr := uint32(r[0])
	if ptr == 0 || ptr >= c.mem.size() {
		panic(oomErr)
	}
	return ptr
}

func (c *Conn) newBytes(b []byte) uint32 {
	if b == nil {
		return 0
	}

	siz := uint32(len(b))
	ptr := c.new(siz)
	buf, ok := c.mem.read(ptr, siz)
	if !ok {
		c.api.free.Call(c.ctx, uint64(ptr))
		panic(rangeErr)
	}

	copy(buf, b)
	return ptr
}

func (c *Conn) newString(s string) uint32 {
	siz := uint32(len(s) + 1)
	ptr := c.new(siz)
	buf, ok := c.mem.read(ptr, siz)
	if !ok {
		c.api.free.Call(c.ctx, uint64(ptr))
		panic(rangeErr)
	}

	buf[len(s)] = 0
	copy(buf, s)
	return ptr
}
