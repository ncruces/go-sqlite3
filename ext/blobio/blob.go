// Package blobio provides an SQL interface to incremental BLOB I/O.
package blobio

import (
	"errors"
	"io"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Register registers the SQL functions:
//
//	readblob(schema, table, column, rowid, offset, n)
//
// Reads n bytes of a blob, starting at offset.
//
//	writeblob(schema, table, column, rowid, offset, data)
//
// Writes data into a blob, at the given offset.
//
//	openblob(schema, table, column, rowid, write, callback, args...)
//
// Opens blobs for reading or writing.
// The callback is invoked for each open blob,
// and must be bound to an [OpenCallback],
// using [sqlite3.BindPointer] or [sqlite3.Pointer].
// The optional args will be passed to the callback,
// along with the [sqlite3.Blob] handle.
//
// https://sqlite.org/c3ref/blob.html
func Register(db *sqlite3.Conn) {
	db.CreateFunction("readblob", 6, 0, readblob)
	db.CreateFunction("writeblob", 6, 0, writeblob)
	db.CreateFunction("openblob", -1, 0, openblob)
}

// OpenCallback is the type for the openblob callback.
type OpenCallback func(*sqlite3.Blob, ...sqlite3.Value) error

func readblob(ctx sqlite3.Context, arg ...sqlite3.Value) {
	blob, err := getAuxBlob(ctx, arg, false)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	_, err = blob.Seek(arg[4].Int64(), io.SeekStart)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	n := arg[5].Int64()
	if n <= 0 {
		return
	}
	buf := make([]byte, n)

	_, err = io.ReadFull(blob, buf)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	ctx.ResultBlob(buf)
	setAuxBlob(ctx, blob, false)
}

func writeblob(ctx sqlite3.Context, arg ...sqlite3.Value) {
	blob, err := getAuxBlob(ctx, arg, true)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	_, err = blob.Seek(arg[4].Int64(), io.SeekStart)
	if err != nil {
		ctx.ResultError(err)
		return
	}

	_, err = blob.Write(arg[5].RawBlob())
	if err != nil {
		ctx.ResultError(err)
		return
	}

	setAuxBlob(ctx, blob, false)
}

func openblob(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if len(arg) < 6 {
		ctx.ResultError(util.ErrorString("openblob: wrong number of arguments"))
		return
	}

	blob, err := getAuxBlob(ctx, arg, arg[4].Bool())
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

	setAuxBlob(ctx, blob, true)
}

func getAuxBlob(ctx sqlite3.Context, arg []sqlite3.Value, write bool) (*sqlite3.Blob, error) {
	row := arg[3].Int64()

	if blob, ok := ctx.GetAuxData(0).(*sqlite3.Blob); ok {
		if err := blob.Reopen(row); errors.Is(err, sqlite3.MISUSE) {
			// Blob was closed (db, table, column or write changed).
		} else {
			return blob, err
		}
	}

	db := arg[0].Text()
	table := arg[1].Text()
	column := arg[2].Text()
	return ctx.Conn().OpenBlob(db, table, column, row, write)
}

func setAuxBlob(ctx sqlite3.Context, blob *sqlite3.Blob, open bool) {
	// This ensures the blob is closed if db, table, column or write change.
	ctx.SetAuxData(0, blob) // db
	ctx.SetAuxData(1, blob) // table
	ctx.SetAuxData(2, blob) // column
	if open {
		ctx.SetAuxData(4, blob) // write
	}
}
