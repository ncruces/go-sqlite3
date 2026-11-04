//go:build freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock || sqlite3_dotlk

package vfs

import (
	"sync"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// +checklocks:s.Mutex
func (s *vfsShm) shmMemLock(offset, n int32, flags _ShmFlag) error {
	switch {
	case flags&_SHM_UNLOCK != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				if s.vfsShmParent.lock[i] <= 0 {
					s.vfsShmParent.lock[i] = 0
				} else {
					s.vfsShmParent.lock[i]--
				}
				s.lock[i] = false
			}
		}
	case flags&_SHM_SHARED != 0:
		for i := offset; i < offset+n; i++ {
			if !s.lock[i] &&
				s.vfsShmParent.lock[i]+1 <= 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			if !s.lock[i] {
				s.vfsShmParent.lock[i]++
				s.lock[i] = true
			}
		}
	case flags&_SHM_EXCLUSIVE != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				// SQLite never requests an exclusive lock that it already holds.
				panic(errutil.AssertErr())
			}
			if s.vfsShmParent.lock[i] != 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmParent.lock[i] = -1
			s.lock[i] = true
		}
	default:
		panic(errutil.AssertErr())
	}
	return nil
}

// vfsShmIOLock implements the I/O lock for
// WAL-indexes shared by all the connections in a process.
// If the WAL-index is also shared with other processes,
// the first reader (or the writer) additionally locks the file,
// and the last one unlocks it; those locks are per process.
type vfsShmIOLock struct {
	rw  sync.RWMutex
	mtx sync.Mutex // held across the file lock calls
	cnt int
}

func (s *vfsShm) IOLock(level LockLevel) (err error) {
	if s.vfsShmParent == nil {
		// The WAL-index is not open.
		return nil
	}

	p := s.vfsShmParent
	l := &p.ioLock

	switch level {
	case LOCK_SHARED:
		l.rw.RLock()
		l.mtx.Lock()
		if l.cnt == 0 {
			err = p.fileIOLock(false)
		}
		if err == nil {
			l.cnt++
			l.mtx.Unlock()
		} else {
			l.mtx.Unlock()
			l.rw.RUnlock()
		}

	case LOCK_EXCLUSIVE:
		l.rw.Lock()
		err = p.fileIOLock(true)
		if err == nil {
			l.cnt = -1
		} else {
			l.rw.Unlock()
		}

	case LOCK_NONE:
		l.mtx.Lock()
		switch {
		case l.cnt < 0:
			l.cnt = 0
			err = p.fileIOUnlock()
			l.mtx.Unlock()
			l.rw.Unlock()
		case l.cnt > 0:
			if l.cnt--; l.cnt == 0 {
				err = p.fileIOUnlock()
			}
			l.mtx.Unlock()
			l.rw.RUnlock()
		default:
			l.mtx.Unlock()
			panic(errutil.AssertErr())
		}

	default:
		panic(errutil.AssertErr())
	}
	return err
}
