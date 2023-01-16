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

	c := Conn{
		module: module,
		memory: module.Memory(),
		api: sqliteAPI{
			malloc:      module.ExportedFunction("malloc"),
			free:        module.ExportedFunction("free"),
			errstr:      module.ExportedFunction("sqlite3_errstr"),
			errmsg:      module.ExportedFunction("sqlite3_errmsg"),
			erroff:      module.ExportedFunction("sqlite3_error_offset"),
			open:        module.ExportedFunction("sqlite3_open_v2"),
			close:       module.ExportedFunction("sqlite3_close"),
			prepare:     module.ExportedFunction("sqlite3_prepare_v3"),
			finalize:    module.ExportedFunction("sqlite3_finalize"),
			exec:        module.ExportedFunction("sqlite3_exec"),
			step:        module.ExportedFunction("sqlite3_step"),
			columnText:  module.ExportedFunction("sqlite3_column_text"),
			columnInt:   module.ExportedFunction("sqlite3_column_int64"),
			columnFloat: module.ExportedFunction("sqlite3_column_double"),
		},
	}

	defer func() {
		if conn == nil {
			c.Close()
		}
	}()

	namePtr := c.newString(name)
	connPtr := c.newBytes(ptrSize)

	r, err := c.api.open.Call(ctx, uint64(namePtr), uint64(connPtr), uint64(flags), 0)
	if err != nil {
		return nil, err
	}

	c.handle, _ = c.memory.ReadUint32Le(connPtr)
	c.free(connPtr)
	c.free(namePtr)

	if r[0] != _OK {
		return nil, c.error(r[0])
	}
	return &c, nil
}

func (c *Conn) Close() error {
	r, err := c.api.close.Call(context.TODO(), uint64(c.handle))
	if err != nil {
		return err
	}

	if r[0] != _OK {
		return c.error(r[0])
	}
	return c.module.Close(context.TODO())
}

func (c *Conn) Exec(sql string) error {
	sqlPtr := c.newString(sql)

	r, err := c.api.exec.Call(context.TODO(), uint64(c.handle), uint64(sqlPtr), 0, 0, 0)
	if err != nil {
		return err
	}

	c.free(sqlPtr)

	if r[0] != _OK {
		return c.error(r[0])
	}
	return nil
}

func (c *Conn) Prepare(sql string) (stmt *Stmt, tail string, err error) {
	return c.PrepareFlags(sql, 0)
}

func (c *Conn) PrepareFlags(sql string, flags PrepareFlag) (stmt *Stmt, tail string, err error) {
	sqlPtr := c.newString(sql)
	stmtPtr := c.newBytes(ptrSize)
	tailPtr := c.newBytes(ptrSize)

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

	c.free(tailPtr)
	c.free(stmtPtr)
	c.free(sqlPtr)

	if r[0] != _OK {
		return nil, "", c.error(r[0])
	}
	if stmt.handle == 0 {
		return nil, "", nil
	}
	return
}

func (c *Conn) error(rc uint64) *Error {
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

	if serr.str == serr.msg {
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

func (c *Conn) newBytes(len uint32) uint32 {
	r, err := c.api.malloc.Call(context.TODO(), uint64(len))
	if err != nil {
		panic(err)
	}
	if r[0] == 0 {
		panic("sqlite3: out of memory")
	}
	return uint32(r[0])
}

func (c *Conn) newString(str string) uint32 {
	ptr := c.newBytes(uint32(len(str) + 1))

	buf, ok := c.memory.Read(ptr, uint32(len(str)+1))
	if !ok {
		c.api.free.Call(context.TODO(), uint64(ptr))
		panic("sqlite3: failed to init string")
	}

	buf[len(str)] = 0
	copy(buf, str)
	return ptr
}

func (c *Conn) getString(ptr, maxlen uint32) string {
	buf, ok := c.memory.Read(ptr, maxlen)
	if !ok {
		if size := c.memory.Size(); ptr < size {
			buf, ok = c.memory.Read(ptr, size-ptr)
		}
		if !ok {
			panic("sqlite3: invalid pointer")
		}
	}
	if i := bytes.IndexByte(buf, 0); i < 0 {
		panic("sqlite3: missing NUL terminator")
	} else {
		return string(buf[:i])
	}
}

const ptrSize = 4

type sqliteAPI struct {
	malloc      api.Function
	free        api.Function
	errstr      api.Function
	errmsg      api.Function
	erroff      api.Function
	open        api.Function
	close       api.Function
	prepare     api.Function
	finalize    api.Function
	exec        api.Function
	step        api.Function
	columnInt   api.Function
	columnText  api.Function
	columnFloat api.Function
}
