// Package readervfs implements an SQLite VFS for immutable databases.
//
// The "reader" [vfs.VFS] permits accessing any [io.ReaderAt]
// as an immutable SQLite database.
//
// Importing package readervfs registers the VFS:
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/readervfs"
package readervfs

import (
	"sync"

	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	vfs.Register("reader", readerVFS{})
}

var (
	readerMtx sync.RWMutex
	// +checklocks:readerMtx
	readerDBs = map[string]ioutil.SizeReaderAt{}
)

// Create creates an immutable database from reader.
// The caller should ensure that data from reader does not mutate,
// otherwise SQLite might return incorrect query results and/or [sqlite3.CORRUPT] errors.
func Create(name string, reader ioutil.SizeReaderAt) {
	readerMtx.Lock()
	readerDBs[name] = reader
	readerMtx.Unlock()
}

// Delete deletes a shared memory database.
func Delete(name string) {
	readerMtx.Lock()
	delete(readerDBs, name)
	readerMtx.Unlock()
}
