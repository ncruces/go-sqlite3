package sqlite3

import (
	"bytes"
	"context"
	"strconv"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Conn struct {
	ctx    context.Context
	handle uint32
	module api.Module
	memory api.Memory
	api    sqliteAPI
}

func Open(filename string) (conn *Conn, err error) {
	return OpenFlags(filename, OPEN_READWRITE|OPEN_CREATE)
}

func OpenFlags(filename string, flags OpenFlag) (conn *Conn, err error) {
	once.Do(compile)

	ctx := context.Background()
	cfg := wazero.NewModuleConfig().
		WithName("sqlite3-" + strconv.FormatUint(counter.Add(1), 10))
	module, err := wasm.InstantiateModule(ctx, module, cfg)
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
	connPtr := c.new(ptrSize)
	defer c.free(namePtr)
	defer c.free(connPtr)

	r, err := c.api.open.Call(c.ctx, uint64(namePtr), uint64(connPtr), uint64(flags), 0)
	if err != nil {
		return nil, err
	}

	c.handle, _ = c.memory.ReadUint32Le(connPtr)

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
	return c.module.Close(c.ctx)
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
	stmtPtr := c.new(ptrSize)
	tailPtr := c.new(ptrSize)
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
	stmt.handle, _ = c.memory.ReadUint32Le(stmtPtr)
	i, _ := c.memory.ReadUint32Le(tailPtr)
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
		err.msg = c.getString(uint32(r[0]), 512)
	}

	r, _ = c.api.errstr.Call(c.ctx, rc)
	if r != nil {
		err.str = c.getString(uint32(r[0]), 512)
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
	if r[0] == 0 {
		panic(oomErr)
	}
	return uint32(r[0])
}

func (c *Conn) newBytes(s []byte) uint32 {
	if s == nil {
		return 0
	}

	siz := uint32(len(s))
	ptr := c.new(siz)
	mem, ok := c.memory.Read(ptr, siz)
	if !ok {
		c.api.free.Call(c.ctx, uint64(ptr))
		panic(rangeErr)
	}

	copy(mem, s)
	return ptr
}

func (c *Conn) newString(s string) uint32 {
	siz := uint32(len(s) + 1)
	ptr := c.new(siz)
	mem, ok := c.memory.Read(ptr, siz)
	if !ok {
		c.api.free.Call(c.ctx, uint64(ptr))
		panic(rangeErr)
	}

	mem[len(s)] = 0
	copy(mem, s)
	return ptr
}

func (c *Conn) getString(ptr, maxlen uint32) string {
	return getString(c.memory, ptr, maxlen)
}

func getString(memory api.Memory, ptr, maxlen uint32) string {
	mem, ok := memory.Read(ptr, maxlen+1)
	if !ok {
		mem, ok = memory.Read(ptr, memory.Size()-ptr)
		if !ok {
			panic(rangeErr)
		}
	}
	if i := bytes.IndexByte(mem, 0); i < 0 {
		panic(noNulErr)
	} else {
		return string(mem[:i])
	}
}

const ptrSize = 4
