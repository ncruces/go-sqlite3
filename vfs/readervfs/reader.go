package readervfs

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

type readerVFS struct{}

func (readerVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	if flags&vfs.OPEN_MAIN_DB == 0 {
		// notest
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
	// notest
	return sqlite3.IOERR_DELETE
}

func (readerVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	// notest
	return false, nil
}

func (readerVFS) FullPathname(name string) (string, error) {
	return name, nil
}

type readerFile struct{ ioutil.SizeReaderAt }

func (readerFile) Close() error {
	return nil
}

func (readerFile) WriteAt(b []byte, off int64) (n int, err error) {
	// notest
	return 0, sqlite3.READONLY
}

func (readerFile) Truncate(size int64) error {
	// notest
	return sqlite3.READONLY
}

func (readerFile) Sync(flag vfs.SyncFlag) error {
	// notest
	return nil
}

func (readerFile) Lock(lock vfs.LockLevel) error {
	// notest
	return nil
}

func (readerFile) Unlock(lock vfs.LockLevel) error {
	// notest
	return nil
}

func (readerFile) CheckReservedLock() (bool, error) {
	// notest
	return false, nil
}

func (readerFile) SectorSize() int {
	// notest
	return 0
}

func (readerFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_IMMUTABLE
}
