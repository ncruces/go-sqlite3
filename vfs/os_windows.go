//go:build !sqlite3_nosys

package vfs

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// SupportsFileLocking is false on platforms that do not support file locking.
// To open a database file in one such platform,
// you need to use the [nolock] or [immutable] URI parameters.
//
// [nolock]: https://sqlite.org/uri.html#urinolock
// [immutable]: https://sqlite.org/uri.html#uriimmutable
const SupportsFileLocking = true

func osGetSharedLock(file *os.File) _ErrorCode {
	// Acquire the PENDING lock temporarily before acquiring a new SHARED lock.
	rc := osReadLock(file, _PENDING_BYTE, 1, 0)

	if rc == _OK {
		// Acquire the SHARED lock.
		rc = osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)

		// Release the PENDING lock.
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return rc
}

func osGetReservedLock(file *os.File) _ErrorCode {
	// Acquire the RESERVED lock.
	return osWriteLock(file, _RESERVED_BYTE, 1, 0)
}

func osGetPendingLock(file *os.File, block bool) _ErrorCode {
	var timeout time.Duration
	if block {
		timeout = -1
	}

	// Acquire the PENDING lock.
	return osWriteLock(file, _PENDING_BYTE, 1, timeout)
}

func osGetExclusiveLock(file *os.File, wait bool) _ErrorCode {
	var timeout time.Duration
	if wait {
		timeout = time.Millisecond
	}

	// Release the SHARED lock.
	osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	rc := osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)

	if rc != _OK {
		// Reacquire the SHARED lock.
		osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)
	}
	return rc
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	if state >= LOCK_EXCLUSIVE {
		// Release the EXCLUSIVE lock.
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

		// Reacquire the SHARED lock.
		if rc := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// This should never happen.
			// We should always be able to reacquire the read lock.
			return _IOERR_RDLOCK
		}
	}

	// Release the PENDING and RESERVED locks.
	if state >= LOCK_RESERVED {
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	if state >= LOCK_PENDING {
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func osReleaseLock(file *os.File, state LockLevel) _ErrorCode {
	// Release all locks.
	if state >= LOCK_RESERVED {
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	if state >= LOCK_SHARED {
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	if state >= LOCK_PENDING {
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	// Test the RESERVED lock.
	return osCheckLock(file, _RESERVED_BYTE, 1)
}

func osUnlock(file *os.File, start, len uint32) _ErrorCode {
	err := windows.UnlockFileEx(windows.Handle(file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err == windows.ERROR_NOT_LOCKED {
		return _OK
	}
	if err != nil {
		return _IOERR_UNLOCK
	}
	return _OK
}

func osLock(file *os.File, flags, start, len uint32, timeout time.Duration, def _ErrorCode) _ErrorCode {
	var err error
	switch {
	case timeout == 0:
		err = osLockEx(file, flags|windows.LOCKFILE_FAIL_IMMEDIATELY, start, len)
	case timeout < 0:
		err = osLockEx(file, flags, start, len)
	default:
		before := time.Now()
		for {
			err = osLockEx(file, flags|windows.LOCKFILE_FAIL_IMMEDIATELY, start, len)
			if errno, _ := err.(windows.Errno); errno != windows.ERROR_LOCK_VIOLATION {
				break
			}
			if timeout <= 0 || timeout < time.Since(before) {
				break
			}
			osSleep(time.Millisecond)
		}
	}
	return osLockErrorCode(err, def)
}

func osLockEx(file *os.File, flags, start, len uint32) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), flags,
		0, len, 0, &windows.Overlapped{Offset: start})
}

func osReadLock(file *os.File, start, len uint32, timeout time.Duration) _ErrorCode {
	return osLock(file, 0, start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len uint32, timeout time.Duration) _ErrorCode {
	return osLock(file, windows.LOCKFILE_EXCLUSIVE_LOCK, start, len, timeout, _IOERR_LOCK)
}

func osCheckLock(file *os.File, start, len uint32) (bool, _ErrorCode) {
	rc := osLock(file, 0, start, len, 0, _IOERR_CHECKRESERVEDLOCK)
	if rc == _BUSY {
		return true, _OK
	}
	if rc == _OK {
		osUnlock(file, start, len)
	}
	return false, rc
}

func osLockErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(windows.Errno); ok {
		// https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
		switch errno {
		case
			windows.ERROR_LOCK_VIOLATION,
			windows.ERROR_IO_PENDING,
			windows.ERROR_OPERATION_ABORTED:
			return _BUSY
		}
	}
	return def
}

func osSleep(d time.Duration) {
	if d > 0 {
		period := max(1, d/(5*time.Millisecond))
		if period < 16 {
			windows.TimeBeginPeriod(uint32(period))
		}
		time.Sleep(d)
		if period < 16 {
			windows.TimeEndPeriod(uint32(period))
		}
	}
}
