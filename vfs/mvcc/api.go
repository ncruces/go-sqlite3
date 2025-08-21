// Package mvcc implements the "mvcc" SQLite VFS.
//
// The "mvcc" [vfs.VFS] allows the same in-memory database to be shared
// among multiple database connections in the same process,
// as long as the database name begins with "/".
//
// Importing package mvcc registers the VFS:
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/mvcc"
package mvcc

import (
	"sync"

	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	vfs.Register("mvcc", mvccVFS{})
}

var (
	memoryMtx sync.Mutex
	// +checklocks:memoryMtx
	memoryDBs = map[string]*mvccDB{}
)

// Create creates a shared memory database,
// using data as its initial contents.
func Create(name string, data string) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()

	db := &mvccDB{
		refs: 1,
		name: name,
	}
	memoryDBs[name] = db
	if len(data) == 0 {
		return
	}
	// Convert data from WAL/2 to rollback journal.
	if len(data) >= 20 && (false ||
		data[18] == 2 && data[19] == 2 ||
		data[18] == 3 && data[19] == 3) {
		db.data = db.data.
			Put(0, data[:18]).
			Put(18, "\001\001").
			Put(20, data[20:])
	} else {
		db.data = db.data.Put(0, data)
	}
}

// Delete deletes a shared memory database.
func Delete(name string) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	delete(memoryDBs, name)
}

// Snapshot stores a snapshot of database src into dst.
func Snapshot(dst, src string) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	memoryDBs[dst] = memoryDBs[src].fork()
}
