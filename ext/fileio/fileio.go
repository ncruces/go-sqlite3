package fileio

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ncruces/go-sqlite3"
)

func Register(db *sqlite3.Conn) {
	db.CreateFunction("lsmode", 1, 0, lsmode)
	db.CreateFunction("readfile", 1, sqlite3.DIRECTONLY, readfile)
	db.CreateFunction("writefile", -1, sqlite3.DIRECTONLY, writefile)
	sqlite3.CreateModule(db, "fsdir", nil, func(db *sqlite3.Conn, module, schema, table string, arg ...string) (fsdir, error) {
		err := db.DeclareVtab(`CREATE TABLE x(name,mode,mtime,data,path HIDDEN,dir HIDDEN)`)
		db.VtabConfig(sqlite3.VTAB_DIRECTONLY)
		return fsdir{}, err
	})
}

func lsmode(ctx sqlite3.Context, arg ...sqlite3.Value) {
	ctx.ResultText(fs.FileMode(arg[0].Int()).String())
}

func readfile(ctx sqlite3.Context, arg ...sqlite3.Value) {
	d, err := os.ReadFile(arg[0].Text())
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		ctx.ResultError(fmt.Errorf("readfile: %w", err))
	} else {
		ctx.ResultBlob(d)
	}
}
