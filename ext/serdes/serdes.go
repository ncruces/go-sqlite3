// Package serdes provides functions to (de)serialize databases.
package serdes

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

const vfsName = "github.com/ncruces/go-sqlite3/ext/serdes.sliceVFS"

func init() {
	vfs.Register(vfsName, sliceVFS{})
}

var fileToOpen = make(chan *[]byte, 1)

// Serialize backs up a database into a byte slice.
//
// https://sqlite.org/c3ref/serialize.html
func Serialize(db *sqlite3.Conn, schema string) ([]byte, error) {
	var file []byte
	fileToOpen <- &file
	err := db.Backup(schema, "file:serdes.db?vfs="+vfsName)
	return file, err
}

// Deserialize restores a database from a byte slice,
// DESTROYING any contents previously stored in schema.
//
// To non-destructively open a database from a byte slice,
// consider alternatives like the ["reader"] or ["memdb"] VFSes.
//
// This differs from the similarly named SQLite API
// in that it DOES NOT disconnect from schema
// to reopen as an in-memory database.
//
// https://sqlite.org/c3ref/deserialize.html
//
// ["memdb"]: https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/memdb
// ["reader"]: https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/readervfs
func Deserialize(db *sqlite3.Conn, schema string, data []byte) error {
	fileToOpen <- &data
	return db.Restore(schema, "file:serdes.db?vfs="+vfsName)
}

type sliceVFS struct{}

func (sliceVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	if flags&vfs.OPEN_MAIN_DB == 0 || name != "serdes.db" {
		return nil, flags, sqlite3.CANTOPEN
	}
	select {
	case file := <-fileToOpen:
		return (*vfsutil.SliceFile)(file), flags | vfs.OPEN_MEMORY, nil
	default:
		return nil, flags, sqlite3.MISUSE
	}
}

func (sliceVFS) Delete(name string, dirSync bool) error {
	// notest // OPEN_MEMORY
	return sqlite3.IOERR_DELETE
}

func (sliceVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	return name == "serdes.db", nil
}

func (sliceVFS) FullPathname(name string) (string, error) {
	return name, nil
}
