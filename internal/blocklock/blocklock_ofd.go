//go:build linux && !(sqlite3_flock || sqlite3_dotlk)

package blocklock

import (
	"github.com/ncruces/go-sqlite3"
	"golang.org/x/sys/unix"
)

// Locks are placed far past the end of any SQLite database,
// so they never overlap the locks SQLite itself takes
// on the main database file.
const lockBase = 1 << 62

func osSupported(fd uintptr) bool {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: lockBase,
		Len:   1,
	}
	return unix.FcntlFlock(fd, unix.F_OFD_GETLK, &lock) == nil
}

func osReadLock(fd uintptr, start, end int64) error {
	return osLock(fd, unix.F_OFD_SETLKW, unix.F_RDLCK, start, end, sqlite3.IOERR_RDLOCK)
}

func osWriteLock(fd uintptr, start, end int64) error {
	return osLock(fd, unix.F_OFD_SETLKW, unix.F_WRLCK, start, end, sqlite3.IOERR_LOCK)
}

func osUnlock(fd uintptr, start, end int64) {
	osLock(fd, unix.F_OFD_SETLK, unix.F_UNLCK, start, end, sqlite3.IOERR_UNLOCK)
}

func osLock(fd uintptr, cmd int, typ int16, start, end int64, def sqlite3.ExtendedErrorCode) error {
	lock := unix.Flock_t{
		Type:  typ,
		Start: lockBase + start,
		Len:   end - start,
	}
	for {
		err := unix.FcntlFlock(fd, cmd, &lock)
		if err == nil {
			return nil
		}
		if err != unix.EINTR {
			return def
		}
	}
}
