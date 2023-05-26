package sqlite3vfs

import (
	"io"
	"io/fs"
)

// A ReaderVFS is a [VFS] for immutable databases.
type ReaderVFS map[string]SizeReaderAt

var _ VFS = ReaderVFS{}

// A SizeReaderAt is a ReaderAt with a Size method.
// Use [NewSizeReaderAt] to adapt different Size interfaces.
type SizeReaderAt interface {
	Size() (int64, error)
	io.ReaderAt
}

// Open implements the [VFS] interface.
func (vfs ReaderVFS) Open(name string, flags OpenFlag) (File, OpenFlag, error) {
	if flags&OPEN_MAIN_DB == 0 {
		return nil, flags, _CANTOPEN
	}
	if ra, ok := vfs[name]; ok {
		return readerFile{ra}, flags, nil
	}
	return nil, flags, _CANTOPEN
}

// Delete implements the [VFS] interface.
func (vfs ReaderVFS) Delete(name string, dirSync bool) error {
	return _IOERR_DELETE
}

// Access implements the [VFS] interface.
func (vfs ReaderVFS) Access(name string, flag AccessFlag) (bool, error) {
	return false, nil
}

// FullPathname implements the [VFS] interface.
func (vfs ReaderVFS) FullPathname(name string) (string, error) {
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
	return 0, _READONLY
}

func (readerFile) Truncate(size int64) error {
	return _READONLY
}

func (readerFile) Sync(flag SyncFlag) error {
	return nil
}

func (readerFile) Lock(lock LockLevel) error {
	return nil
}

func (readerFile) Unlock(lock LockLevel) error {
	return nil
}

func (readerFile) CheckReservedLock() (bool, error) {
	return false, nil
}

func (readerFile) SectorSize() int {
	return 0
}

func (readerFile) DeviceCharacteristics() DeviceCharacteristic {
	return IOCAP_IMMUTABLE
}

// NewSizeReaderAt returns a SizeReaderAt given an io.ReaderAt
// that implements one of:
//   - Size() (int64, error)
//   - Size() int64
//   - Len() int
//   - Stat() (fs.FileInfo, error)
//   - Seek(offset int64, whence int) (int64, error)
func NewSizeReaderAt(r io.ReaderAt) SizeReaderAt {
	return sizer{r}
}

type sizer struct{ io.ReaderAt }

func (s sizer) Size() (int64, error) {
	switch s := s.ReaderAt.(type) {
	case interface{ Size() (int64, error) }:
		return s.Size()
	case interface{ Size() int64 }:
		return s.Size(), nil
	case interface{ Len() int }:
		return int64(s.Len()), nil
	case interface{ Stat() (fs.FileInfo, error) }:
		fi, err := s.Stat()
		if err != nil {
			return 0, err
		}
		return fi.Size(), nil
	case io.Seeker:
		return s.Seek(0, io.SeekEnd)
	}
	return 0, _IOERR_SEEK
}
