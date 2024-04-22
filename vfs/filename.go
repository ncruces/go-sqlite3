package vfs

import (
	"context"
	"net/url"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

// Filename is used by SQLite to pass filenames
// to the Open method of a VFS.
//
// https://sqlite.org/c3ref/filename.html
type Filename struct {
	ctx   context.Context
	mod   api.Module
	zPath uint32
	flags OpenFlag
	stack [2]uint64
}

// OpenFilename is an internal API users should not call directly.
func OpenFilename(ctx context.Context, mod api.Module, id uint32, flags OpenFlag) *Filename {
	if id == 0 {
		return nil
	}
	return &Filename{
		ctx:   ctx,
		mod:   mod,
		zPath: id,
		flags: flags,
	}
}

// String returns this filename as a string.
func (n *Filename) String() string {
	if n == nil || n.zPath == 0 {
		return ""
	}
	return util.ReadString(n.mod, n.zPath, _MAX_PATHNAME)
}

// Database returns the name of the corresponding database file.
//
// https://sqlite.org/c3ref/filename_database.html
func (n *Filename) Database() string {
	return n.path("sqlite3_filename_database")
}

// Journal returns the name of the corresponding rollback journal file.
//
// https://sqlite.org/c3ref/filename_database.html
func (n *Filename) Journal() string {
	return n.path("sqlite3_filename_journal")
}

// Journal returns the name of the corresponding WAL file.
//
// https://sqlite.org/c3ref/filename_database.html
func (n *Filename) WAL() string {
	return n.path("sqlite3_filename_wal")
}

func (n *Filename) path(method string) string {
	if n == nil || n.zPath == 0 {
		return ""
	}
	n.stack[0] = uint64(n.zPath)
	fn := n.mod.ExportedFunction(method)
	if err := fn.CallWithStack(n.ctx, n.stack[:]); err != nil {
		panic(err)
	}
	return util.ReadString(n.mod, uint32(n.stack[0]), _MAX_PATHNAME)
}

// DatabaseFile returns the main database [File] corresponding to a journal.
//
// https://sqlite.org/c3ref/database_file_object.html
func (n *Filename) DatabaseFile() File {
	if n == nil || n.zPath == 0 {
		return nil
	}
	if n.flags&(OPEN_MAIN_DB|OPEN_MAIN_JOURNAL|OPEN_WAL) == 0 {
		return nil
	}

	n.stack[0] = uint64(n.zPath)
	fn := n.mod.ExportedFunction("sqlite3_database_file_object")
	if err := fn.CallWithStack(n.ctx, n.stack[:]); err != nil {
		panic(err)
	}
	file, _ := vfsFileGet(n.ctx, n.mod, uint32(n.stack[0])).(File)
	return file
}

// URIParameters obtain values for URI parameters.
//
// https://sqlite.org/c3ref/uri_boolean.html
func (n *Filename) URIParameters() url.Values {
	if n == nil || n.zPath == 0 {
		return nil
	}

	var params url.Values
	uriKey := n.mod.ExportedFunction("sqlite3_uri_key")
	uriParam := n.mod.ExportedFunction("sqlite3_uri_parameter")

	for i := 0; ; i++ {
		n.stack[1] = uint64(i)
		n.stack[0] = uint64(n.zPath)
		if err := uriKey.CallWithStack(n.ctx, n.stack[:]); err != nil {
			panic(err)
		}
		if n.stack[0] == 0 {
			return params
		}
		key := util.ReadString(n.mod, uint32(n.stack[0]), _MAX_NAME)

		n.stack[1] = n.stack[0]
		n.stack[0] = uint64(n.zPath)
		if err := uriParam.CallWithStack(n.ctx, n.stack[:]); err != nil {
			panic(err)
		}
		if params == nil {
			params = url.Values{}
		}
		params.Add(key, util.ReadString(n.mod, uint32(n.stack[0]), _MAX_NAME))
	}
}
