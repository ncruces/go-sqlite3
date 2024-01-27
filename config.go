package sqlite3

import (
	"context"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

// Config makes configuration changes to a database connection.
// Only boolean configuration options are supported.
// Called with no arg reads the current configuration value,
// called with one arg sets and returns the new value.
//
// https://sqlite.org/c3ref/db_config.html
func (c *Conn) Config(op DBConfig, arg ...bool) (bool, error) {
	defer c.arena.mark()()
	argsPtr := c.arena.new(2 * ptrlen)

	var flag int
	switch {
	case len(arg) == 0:
		flag = -1
	case arg[0]:
		flag = 1
	}

	util.WriteUint32(c.mod, argsPtr+0*ptrlen, uint32(flag))
	util.WriteUint32(c.mod, argsPtr+1*ptrlen, argsPtr)

	r := c.call("sqlite3_db_config", uint64(c.handle),
		uint64(op), uint64(argsPtr))
	return util.ReadUint32(c.mod, argsPtr) != 0, c.error(r)
}

// ConfigLog sets up the error logging callback for the connection.
//
// https://sqlite.org/errlog.html
func (c *Conn) ConfigLog(cb func(code ExtendedErrorCode, msg string)) error {
	var enable uint64
	if cb != nil {
		enable = 1
	}
	r := c.call("sqlite3_config_log_go", enable)
	if err := c.error(r); err != nil {
		return err
	}
	c.log = cb
	return nil
}

func logCallback(ctx context.Context, mod api.Module, _, iCode, zMsg uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.log != nil {
		msg := util.ReadString(mod, zMsg, _MAX_LENGTH)
		c.log(xErrorCode(iCode), msg)
	}
}

// Limit allows the size of various constructs to be
// limited on a connection by connection basis.
//
// https://sqlite.org/c3ref/limit.html
func (c *Conn) Limit(id LimitCategory, value int) int {
	r := c.call("sqlite3_limit", uint64(c.handle), uint64(id), uint64(value))
	return int(int32(r))
}

func authorizerCallback(ctx context.Context, mod api.Module, pDB, action, zName3rd, zName4th, zSchema, zInnerName uint32) uint32 {
	return 0
}
