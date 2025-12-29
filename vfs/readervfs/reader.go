package readervfs

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

type readerVFS struct{}

func (readerVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// Temporary files use the default VFS.
	if name == "" || flags&vfs.OPEN_DELETEONCLOSE != 0 {
		return vfs.Find("").Open(name, flags)
	}
	// Refuse to open all other file types.
	if flags&vfs.OPEN_MAIN_DB == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}
	readerMtx.RLock()
	defer readerMtx.RUnlock()
	if ra, ok := readerDBs[name]; ok {
		return readerFile{ra}, flags | vfs.OPEN_READONLY, nil
	}
	return nil, flags, sqlite3.CANTOPEN
}

func (readerVFS) Delete(name string, dirSync bool) error {
	// notest // IOCAP_IMMUTABLE
	return sqlite3.IOERR_DELETE
}

func (readerVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	// notest // IOCAP_IMMUTABLE
	return false, sqlite3.IOERR_ACCESS
}

func (readerVFS) FullPathname(name string) (string, error) {
	return name, nil
}

type readerFile struct{ ioutil.SizeReaderAt }

func (readerFile) Close() error {
	return nil
}

func (readerFile) WriteAt(b []byte, off int64) (n int, err error) {
	// notest // IOCAP_IMMUTABLE
	return 0, sqlite3.IOERR_WRITE
}

func (readerFile) Truncate(size int64) error {
	// notest // IOCAP_IMMUTABLE
	return sqlite3.IOERR_TRUNCATE
}

func (readerFile) Sync(flag vfs.SyncFlag) error {
	// notest // IOCAP_IMMUTABLE
	return sqlite3.IOERR_FSYNC
}

func (readerFile) Lock(lock vfs.LockLevel) error {
	// notest // IOCAP_IMMUTABLE
	return sqlite3.IOERR_LOCK
}

func (readerFile) Unlock(lock vfs.LockLevel) error {
	// notest // IOCAP_IMMUTABLE
	return sqlite3.IOERR_UNLOCK
}

func (readerFile) CheckReservedLock() (bool, error) {
	// notest // IOCAP_IMMUTABLE
	return false, sqlite3.IOERR_CHECKRESERVEDLOCK
}

func (readerFile) SectorSize() int {
	// notest // IOCAP_IMMUTABLE
	return 0
}

func (readerFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_IMMUTABLE | vfs.IOCAP_SUBPAGE_READ
}
