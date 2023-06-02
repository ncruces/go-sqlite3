// Package readervfs implements an SQLite VFS for immutable databases.
//
// The "reader" [vfs.VFS] permits accessing any [io.ReaderAt]
// as an immutable SQLite database.
//
// Importing package readervfs registers the VFS.
//
//	import _ "github.com/ncruces/go-sqlite3/vfs/readervfs"
package readervfs

import (
	"io"
	"io/fs"
	"sync"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	vfs.Register("reader", readerVFS{})
}

var (
	readerMtx sync.RWMutex
	// +checklocks:readerMtx
	readerDBs = map[string]SizeReaderAt{}
)

// Create creates an immutable database from reader.
// The caller should insure that data from reader does not mutate,
// otherwise SQLite might return incorrect query results and/or [sqlite3.CORRUPT] errors.
func Create(name string, reader SizeReaderAt) {
	readerMtx.Lock()
	defer readerMtx.Unlock()
	readerDBs[name] = reader
}

// Delete deletes a shared memory database.
func Delete(name string) {
	readerMtx.Lock()
	defer readerMtx.Unlock()
	delete(readerDBs, name)
}

// A SizeReaderAt is a ReaderAt with a Size method.
// Use [NewSizeReaderAt] to adapt different Size interfaces.
type SizeReaderAt interface {
	Size() (int64, error)
	io.ReaderAt
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
	return 0, sqlite3.IOERR_SEEK
}
