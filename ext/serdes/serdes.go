// Package serdes provides functions to (de)serialize databases.
package serdes

import (
	"io"
	"sync"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	vfs.Register(vfsName, sliceVFS{})
}

// Serialize backs up a database into a byte slice.
//
// https://sqlite.org/c3ref/serialize.html
func Serialize(db *sqlite3.Conn, schema string) ([]byte, error) {
	var file sliceFile
	openMtx.Lock()
	openFile = &file
	err := db.Backup(schema, "file:db?vfs="+vfsName)
	return file.data, err
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
	openMtx.Lock()
	openFile = &sliceFile{data}
	return db.Restore(schema, "file:db?vfs="+vfsName)
}

var (
	openMtx  sync.Mutex
	openFile *sliceFile
)

const vfsName = "github.com/ncruces/go-sqlite3/ext/deserialize.sliceVFS"

type sliceVFS struct{}

func (sliceVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	if flags&vfs.OPEN_MAIN_DB == 0 {
		// notest // OPEN_MEMORY
		return nil, flags, sqlite3.CANTOPEN
	}

	file := openFile
	openFile = nil
	openMtx.Unlock()

	if file.data != nil {
		flags |= vfs.OPEN_READONLY
	}
	flags |= vfs.OPEN_MEMORY
	return file, flags, nil
}

func (sliceVFS) Delete(name string, dirSync bool) error {
	// notest // OPEN_MEMORY
	return sqlite3.IOERR_DELETE
}

func (sliceVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	return name == "db", nil
}

func (sliceVFS) FullPathname(name string) (string, error) {
	return name, nil
}

type sliceFile struct{ data []byte }

func (f *sliceFile) ReadAt(b []byte, off int64) (n int, err error) {
	if d := f.data; off < int64(len(d)) {
		n = copy(b, d[off:])
	}
	if n == 0 {
		err = io.EOF
	}
	return
}

func (f *sliceFile) WriteAt(b []byte, off int64) (n int, err error) {
	if d := f.data; off > int64(len(d)) {
		f.data = append(d, make([]byte, off-int64(len(d)))...)
	}
	d := append(f.data[:off], b...)
	if len(d) > len(f.data) {
		f.data = d
	}
	return len(b), nil
}

func (f *sliceFile) Size() (int64, error) {
	return int64(len(f.data)), nil
}

func (f *sliceFile) Truncate(size int64) error {
	if d := f.data; size < int64(len(d)) {
		f.data = d[:size]
	}
	return nil
}

func (f *sliceFile) SizeHint(size int64) error {
	if d := f.data; size > int64(len(d)) {
		f.data = append(d, make([]byte, size-int64(len(d)))...)
	}
	return nil
}

func (*sliceFile) Close() error { return nil }

func (*sliceFile) Sync(flag vfs.SyncFlag) error { return nil }

func (*sliceFile) Lock(lock vfs.LockLevel) error { return nil }

func (*sliceFile) Unlock(lock vfs.LockLevel) error { return nil }

func (*sliceFile) CheckReservedLock() (bool, error) {
	// notest // OPEN_MEMORY
	return false, nil
}

func (*sliceFile) SectorSize() int {
	// notest // IOCAP_POWERSAFE_OVERWRITE
	return 0
}

func (*sliceFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_ATOMIC |
		vfs.IOCAP_SAFE_APPEND |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_POWERSAFE_OVERWRITE |
		vfs.IOCAP_SUBPAGE_READ
}
