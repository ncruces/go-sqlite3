package sqlite3reader

import (
	"io"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

type vfs struct{}

// Open implements the [sqlite3vfs.VFS] interface.
func (vfs) Open(name string, flags sqlite3vfs.OpenFlag) (sqlite3vfs.File, sqlite3vfs.OpenFlag, error) {
	if flags&sqlite3vfs.OPEN_MAIN_DB == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}
	readerMtx.RLock()
	defer readerMtx.RUnlock()
	if ra, ok := readerDBs[name]; ok {
		return readerFile{ra}, flags | sqlite3vfs.OPEN_READONLY, nil
	}
	return nil, flags, sqlite3.CANTOPEN
}

// Delete implements the [sqlite3vfs.VFS] interface.
func (vfs) Delete(name string, dirSync bool) error {
	return sqlite3.IOERR_DELETE
}

// Access implements the [sqlite3vfs.VFS] interface.
func (vfs) Access(name string, flag sqlite3vfs.AccessFlag) (bool, error) {
	return false, nil
}

// FullPathname implements the [sqlite3vfs.VFS] interface.
func (vfs) FullPathname(name string) (string, error) {
	return name, nil
}

type readerFile struct{ SizeReaderAt }

func (r readerFile) Close() error {
	if c, ok := r.SizeReaderAt.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (readerFile) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, sqlite3.READONLY
}

func (readerFile) Truncate(size int64) error {
	return sqlite3.READONLY
}

func (readerFile) Sync(flag sqlite3vfs.SyncFlag) error {
	return nil
}

func (readerFile) Lock(lock sqlite3vfs.LockLevel) error {
	return nil
}

func (readerFile) Unlock(lock sqlite3vfs.LockLevel) error {
	return nil
}

func (readerFile) CheckReservedLock() (bool, error) {
	return false, nil
}

func (readerFile) SectorSize() int {
	return 0
}

func (readerFile) DeviceCharacteristics() sqlite3vfs.DeviceCharacteristic {
	return sqlite3vfs.IOCAP_IMMUTABLE
}
