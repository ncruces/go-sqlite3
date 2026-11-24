//go:build !sqlite3_dotlk

package vfs

import (
	"io"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

const _WALINDEX_PGSZ = 32768

type vfsShm struct {
	*os.File
	path     string
	regions  []*sqlite3_wrap.MappedRegion
	fileLock bool
	blocking bool
}

var _ blockingSharedMemory = &vfsShm{}

func (s *vfsShm) shmOpen() error {
	if s.fileLock {
		return nil
	}
	if s.File == nil {
		f, err := os.OpenFile(s.path,
			os.O_RDWR|os.O_CREATE|windows.O_FILE_FLAG_OVERLAPPED, 0666)
		if err != nil {
			return sysError{err, _CANTOPEN}
		}
		s.fileLock = false
		s.File = f
	}

	// Dead man's switch.
	if osWriteLock(s.File, _SHM_DMS, 1, 0) == nil {
		err := s.Truncate(0)
		osUnlock(s.File, _SHM_DMS, 1)
		if err != nil {
			return sysError{err, _IOERR_SHMOPEN}
		}
	}
	err := osReadLock(s.File, _SHM_DMS, 1, time.Millisecond)
	s.fileLock = err == nil
	return err
}

func (s *vfsShm) shmMap(wrp *sqlite3_wrap.Wrapper, id, size int32, extend bool) (_ ptr_t, err error) {
	// Ensure pages are the expected size, and that we can map files.
	if size != _WALINDEX_PGSZ || !wrp.CanMapFiles() {
		return 0, _IOERR_SHMMAP
	}

	if err := s.shmOpen(); err != nil {
		return 0, err
	}

	// Check if file is big enough.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, sysError{err, _IOERR_SHMSIZE}
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, nil
		}
		if err := osAllocate(s.File, n); err != nil {
			return 0, sysError{err, _IOERR_SHMSIZE}
		}
	}

	r, err := wrp.MapRegion(s.File, int64(id)*int64(size), size, false)
	if err != nil {
		return 0, err
	}
	if r == nil {
		return 0, _IOERR_NOMEM
	}
	s.regions = append(s.regions, r)
	return r.Ptr, nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) error {
	if s.File == nil {
		return _IOERR_SHMLOCK
	}

	var timeout time.Duration
	if s.blocking {
		timeout = time.Millisecond
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(s.File, _SHM_BASE+uint32(offset), uint32(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(s.File, _SHM_BASE+uint32(offset), uint32(n), timeout)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(s.File, _SHM_BASE+uint32(offset), uint32(n), timeout)
	default:
		panic(errutil.AssertErr())
	}
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.File == nil {
		return
	}

	// Unmap regions.
	for _, r := range s.regions {
		r.Unmap()
	}
	s.regions = nil

	// Close the file.
	s.Close()
	s.File = nil
	s.fileLock = false
	if delete {
		os.Remove(s.path)
	}
}

func (s *vfsShm) shmBarrier() {
	var b atomic.Bool
	b.Swap(true)
}

func (s *vfsShm) shmEnableBlocking(block bool) {
	s.blocking = block
}

func (s *vfsShm) IOLock(lock LockLevel, id int64) error {
	if !s.fileLock {
		// The WAL-index is not open.
		return nil
	}

	off := _SHM_IOLK + uint32(uint64(id)%_SHM_NIOLK)

	switch lock {
	case LOCK_NONE:
		return osUnlock(s.File, off, 1)
	case LOCK_SHARED:
		return osLockWait(s.File, 0, off, 1, _IOERR_RDLOCK)
	case LOCK_EXCLUSIVE:
		return osLockWait(s.File, windows.LOCKFILE_EXCLUSIVE_LOCK, off, 1, _IOERR_LOCK)
	default:
		panic(errutil.AssertErr())
	}
}
