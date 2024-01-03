package ioutil

import (
	"io"
	"io/fs"

	"github.com/ncruces/go-sqlite3"
)

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
