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
// The new database retains data,
// but never modifies it.
// The caller may reuse data,
// but must not modify it after this call.
func Create(name string, data []byte) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()

	db := &mvccDB{
		refs: 1,
		name: name,
		size: int64(len(data)),
	}

	// Convert data from WAL/2 to rollback journal.
	if len(data) >= 20 && (false ||
		data[18] == 2 && data[19] == 2 ||
		data[18] == 3 && data[19] == 3) {
		data[18] = 1
		data[19] = 1
	}

	for i := range divRoundUp(db.size, sectorSize) {
		db.data = db.data.Put(i, data[i*sectorSize:])
	}

	memoryDBs[name] = db
}

// Delete deletes a shared memory database.
func Delete(name string) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	delete(memoryDBs, name)
}

// Snapshot stores a snapshot of databases src into dst.
func Snapshot(dst, src string) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	memoryDBs[dst] = memoryDBs[src].fork()
}
