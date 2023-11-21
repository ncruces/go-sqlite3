// Package blob provides an alternative interface to incremental BLOB I/O.
package blob

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Register registers the blob_open SQL function:
//
//	blob_open(schema, table, column, rowid, flags, callback, args...)
//
// The callback must be an [sqlite3.Pointer] to an [OpenCallback].
// Any optional args will be passed to the callback,
// along with the [sqlite3.Blob] handle.
//
// https://sqlite.org/c3ref/blob.html
func Register(db *sqlite3.Conn) {
	db.CreateFunction("blob_open", -1,
		sqlite3.DETERMINISTIC|sqlite3.DIRECTONLY, openBlob)
}

func openBlob(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if len(arg) < 6 {
		ctx.ResultError(util.ErrorString("wrong number of arguments to function blob_open()"))
		return
	}

	row := arg[3].Int64()

	var err error
	blob, ok := ctx.GetAuxData(0).(*sqlite3.Blob)
	if ok {
		err = blob.Reopen(row)
		if errors.Is(err, sqlite3.MISUSE) {
			// Blob was closed (db, table, column or write changed).
			ok = false
		}
	}

	if !ok {
		db := arg[0].Text()
		table := arg[1].Text()
		column := arg[2].Text()
		write := arg[4].Bool()
		blob, err = ctx.Conn().OpenBlob(db, table, column, row, write)
	}
	if err != nil {
		ctx.ResultError(err)
		return
	}

	fn := arg[5].Pointer().(OpenCallback)
	err = fn(blob, arg[6:]...)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	// This ensures the blob is closed if db, table, column or write change.
	ctx.SetAuxData(0, blob)
	ctx.SetAuxData(1, blob)
	ctx.SetAuxData(2, blob)
	ctx.SetAuxData(4, blob)
}

// OpenCallback is the type for the blob_open callback.
type OpenCallback func(*sqlite3.Blob, ...sqlite3.Value) error
