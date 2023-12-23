package sqlite3

import "github.com/ncruces/go-sqlite3/internal/util"

// Config makes configuration changes to a database connection.
// Only bool configuratiton options are supported.
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
