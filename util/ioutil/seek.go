package ioutil

import (
	"io"
	"sync"
)

// SeekingReaderAt implements [io.ReaderAt]
// through an underlying [io.ReadSeeker].
type SeekingReaderAt struct {
	r io.ReadSeeker
	l sync.Mutex
}

// NewSeekingReaderAt creates a new SeekingReaderAt.
// The SeekingReaderAt takes ownership of r
// and will modify its seek offset,
// so callers should not use r after this call.
func NewSeekingReaderAt(r io.ReadSeeker) *SeekingReaderAt {
	return &SeekingReaderAt{r: r}
}

// ReadAt implements [io.ReaderAt].
func (s *SeekingReaderAt) ReadAt(p []byte, off int64) (n int, _ error) {
	s.l.Lock()
	defer s.l.Unlock()

	_, err := s.r.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}

	for len(p) > 0 {
		i, err := s.r.Read(p)
		p = p[i:]
		n += i
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Size implements [SizeReaderAt].
func (s *SeekingReaderAt) Size() (int64, error) {
	s.l.Lock()
	defer s.l.Unlock()
	return s.r.Seek(0, io.SeekEnd)
}

// ReadAt implements [io.Closer].
func (s *SeekingReaderAt) Close() error {
	s.l.Lock()
	defer s.l.Unlock()
	if c, ok := s.r.(io.Closer); ok {
		s.r = nil
		return c.Close()
	}
	return nil
}
