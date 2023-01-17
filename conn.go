package sqlite3

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"strconv"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/writefs"
)

type Conn struct {
	handle uint32
	module api.Module
	memory api.Memory
	api    sqliteAPI
}

func Open(name string) (conn *Conn, err error) {
	return OpenFlags(name, OPEN_READWRITE|OPEN_CREATE)
}

func OpenFlags(name string, flags OpenFlag) (conn *Conn, err error) {
	once.Do(compile)

	var fs fs.FS
	if name != ":memory:" {
		dir := filepath.Dir(name)
		name = filepath.Base(name)
		fs, err = writefs.NewDirFS(dir)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.TODO()

	cfg := wazero.NewModuleConfig().
		WithName("sqlite3-" + strconv.FormatUint(counter.Add(1), 10))
	if fs != nil {
		cfg = cfg.WithFS(fs)
	}
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
	namePtr := c.newString(name)
	connPtr := c.new(ptrSize)
	defer c.free(namePtr)
	defer c.free(connPtr)

	r, err := c.api.open.Call(ctx, uint64(namePtr), uint64(connPtr), uint64(flags), 0)
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
	r, err := c.api.close.Call(context.TODO(), uint64(c.handle))
	if err != nil {
		return err
	}

	if err := c.error(r[0]); err != nil {
		return err
	}
	return c.module.Close(context.TODO())
}

func (c *Conn) Exec(sql string) error {
	sqlPtr := c.newString(sql)
	defer c.free(sqlPtr)

	r, err := c.api.exec.Call(context.TODO(), uint64(c.handle), uint64(sqlPtr), 0, 0, 0)
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

	r, err := c.api.prepare.Call(context.TODO(), uint64(c.handle),
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

	serr := Error{
		Code:         ErrorCode(rc & 0xFF),
		ExtendedCode: ExtendedErrorCode(rc),
	}

	var r []uint64

	// string
	r, _ = c.api.errstr.Call(context.TODO(), rc)
	if r != nil {
		serr.str = c.getString(uint32(r[0]), 512)
	}

	// message
	r, _ = c.api.errmsg.Call(context.TODO(), uint64(c.handle))
	if r != nil {
		serr.msg = c.getString(uint32(r[0]), 512)
	}

	switch serr.msg {
	case "not an error", serr.str:
		serr.msg = ""
	}

	return &serr
}

func (c *Conn) free(ptr uint32) {
	if ptr == 0 {
		return
	}
	_, err := c.api.free.Call(context.TODO(), uint64(ptr))
	if err != nil {
		panic(err)
	}
}

func (c *Conn) new(len uint32) uint32 {
	r, err := c.api.malloc.Call(context.TODO(), uint64(len))
	if err != nil {
		panic(err)
	}
	if r[0] == 0 {
		panic("sqlite3: out of memory")
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
		c.api.free.Call(context.TODO(), uint64(ptr))
		panic("sqlite3: out of range")
	}

	copy(mem, s)
	return ptr
}

func (c *Conn) newString(s string) uint32 {
	siz := uint32(len(s) + 1)
	ptr := c.new(siz)
	mem, ok := c.memory.Read(ptr, siz)
	if !ok {
		c.api.free.Call(context.TODO(), uint64(ptr))
		panic("sqlite3: out of range")
	}

	mem[len(s)] = 0
	copy(mem, s)
	return ptr
}

func (c *Conn) getString(ptr, maxlen uint32) string {
	mem, ok := c.memory.Read(ptr, maxlen)
	if !ok {
		if size := c.memory.Size(); ptr < size {
			mem, ok = c.memory.Read(ptr, size-ptr)
		}
		if !ok {
			panic("sqlite3: out of range")
		}
	}
	if i := bytes.IndexByte(mem, 0); i < 0 {
		panic("sqlite3: missing NUL terminator")
	} else {
		return string(mem[:i])
	}
}

const ptrSize = 4
