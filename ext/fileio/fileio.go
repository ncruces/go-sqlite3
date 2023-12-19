// Package fileio provides SQL functions to read, write and list files.
//
// https://sqlite.org/src/doc/tip/ext/misc/fileio.c
package fileio

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ncruces/go-sqlite3"
)

// Register registers SQL functions readfile, writefile, lsmode,
// and the eponymous virtual table fsdir.
func Register(db *sqlite3.Conn) {
	RegisterFS(db, nil)
}

// Register registers SQL functions readfile, lsmode,
// and the eponymous virtual table fsdir;
// fsys will be used to read files and list directories.
func RegisterFS(db *sqlite3.Conn, fsys fs.FS) {
	db.CreateFunction("lsmode", 1, 0, lsmode)
	db.CreateFunction("readfile", 1, sqlite3.DIRECTONLY, readfile(fsys))
	if fsys == nil {
		db.CreateFunction("writefile", -1, sqlite3.DIRECTONLY, writefile)
	}
	sqlite3.CreateModule(db, "fsdir", nil, func(db *sqlite3.Conn, module, schema, table string, arg ...string) (fsdir, error) {
		err := db.DeclareVtab(`CREATE TABLE x(name,mode,mtime,data,path HIDDEN,dir HIDDEN)`)
		db.VtabConfig(sqlite3.VTAB_DIRECTONLY)
		return fsdir{fsys}, err
	})
}

func lsmode(ctx sqlite3.Context, arg ...sqlite3.Value) {
	ctx.ResultText(fs.FileMode(arg[0].Int()).String())
}

func readfile(fsys fs.FS) func(ctx sqlite3.Context, arg ...sqlite3.Value) {
	return func(ctx sqlite3.Context, arg ...sqlite3.Value) {
		var err error
		var data []byte

		if fsys != nil {
			data, err = fs.ReadFile(fsys, arg[0].Text())
		} else {
			data, err = os.ReadFile(arg[0].Text())
		}

		switch {
		case err == nil:
			ctx.ResultBlob(data)
		case !errors.Is(err, fs.ErrNotExist):
			ctx.ResultError(fmt.Errorf("readfile: %w", err))
		}
	}
}
