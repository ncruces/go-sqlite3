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
// and the table-valued function fsdir.
func Register(db *sqlite3.Conn) error {
	return RegisterFS(db, nil)
}

// Register registers SQL functions readfile, lsmode,
// and the table-valued function fsdir;
// fsys will be used to read files and list directories.
func RegisterFS(db *sqlite3.Conn, fsys fs.FS) error {
	var err error
	if fsys == nil {
		err = db.CreateFunction("writefile", -1, sqlite3.DIRECTONLY, writefile)
	}
	return errors.Join(err,
		db.CreateFunction("readfile", 1, sqlite3.DIRECTONLY, readfile(fsys)),
		db.CreateFunction("lsmode", 1, sqlite3.DETERMINISTIC, lsmode),
		sqlite3.CreateModule(db, "fsdir", nil, func(db *sqlite3.Conn, _, _, _ string, _ ...string) (fsdir, error) {
			err := db.DeclareVTab(`CREATE TABLE x(name,mode,mtime TIMESTAMP,data,path HIDDEN,dir HIDDEN)`)
			db.VTabConfig(sqlite3.VTAB_DIRECTONLY)
			return fsdir{fsys}, err
		}))
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
			ctx.ResultError(fmt.Errorf("readfile: %w", err)) // notest
		}
	}
}
