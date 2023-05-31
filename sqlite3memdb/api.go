// Package sqlite3memdb implements the "memdb" SQLite VFS.
//
// The "memdb" [sqlite3vfs.VFS] allows the same in-memory database to be shared
// among multiple database connections in the same process,
// as long as the database name begins with "/".
//
// Importing package sqlite3memdb registers the VFS.
//
//	import _ "github.com/ncruces/go-sqlite3/sqlite3memdb"
package sqlite3memdb

import (
	"sync"

	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

func init() {
	sqlite3vfs.Register("memdb", vfs{})
}

var (
	memoryMtx sync.Mutex
	memoryDBs = map[string]*dbase{}
)

// Create creates a shared memory database,
// using data as its initial contents.
// The new database takes ownership of data,
// and the caller should not use data after this call.
func Create(name string, data []byte) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()

	db := new(dbase)
	db.size = int64(len(data))

	sectors := divRoundUp(db.size, sectorSize)
	db.data = make([]*[sectorSize]byte, sectors)
	for i := range db.data {
		sector := data[i*sectorSize:]
		if len(sector) >= sectorSize {
			db.data[i] = (*[sectorSize]byte)(sector)
		} else {
			db.data[i] = new([sectorSize]byte)
			copy((*db.data[i])[:], sector)
		}
	}

	memoryDBs[name] = db
}

// Delete deletes a shared memory database.
func Delete(name string) {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	delete(memoryDBs, name)
}
