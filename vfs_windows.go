package sqlite3

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func (vfsOSMethods) DeleteOnClose(file *os.File) {}

func (vfsOSMethods) GetExclusiveLock(file *os.File) xErrorCode {
	// Release the SHARED lock.
	vfsOS.unlock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	rc := vfsOS.writeLock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Reacquire the SHARED lock.
	if rc != _OK {
		vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	return rc
}

func (vfsOSMethods) DowngradeLock(file *os.File, state vfsLockState) xErrorCode {
	if state >= _EXCLUSIVE_LOCK {
		// Release the SHARED lock.
		vfsOS.unlock(file, _SHARED_FIRST, _SHARED_SIZE)

		// Reacquire the SHARED lock.
		if rc := vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE); rc != _OK {
			// This should never happen.
			// We should always be able to reacquire the read lock.
			return IOERR_RDLOCK
		}
	}

	// Release the PENDING and RESERVED locks.
	if state >= _RESERVED_LOCK {
		vfsOS.unlock(file, _RESERVED_BYTE, 1)
	}
	if state >= _PENDING_LOCK {
		vfsOS.unlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func (vfsOSMethods) ReleaseLock(file *os.File, state vfsLockState) xErrorCode {
	// Release all locks.
	if state >= _RESERVED_LOCK {
		vfsOS.unlock(file, _RESERVED_BYTE, 1)
	}
	if state >= _SHARED_LOCK {
		vfsOS.unlock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	if state >= _PENDING_LOCK {
		vfsOS.unlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func (vfsOSMethods) unlock(file *os.File, start, len uint32) xErrorCode {
	err := windows.UnlockFileEx(windows.Handle(file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err == windows.ERROR_NOT_LOCKED {
		return _OK
	}
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) readLock(file *os.File, start, len uint32) xErrorCode {
	return vfsOS.lockErrorCode(windows.LockFileEx(windows.Handle(file.Fd()),
		windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, len, 0, &windows.Overlapped{Offset: start}),
		IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len uint32) xErrorCode {
	return vfsOS.lockErrorCode(windows.LockFileEx(windows.Handle(file.Fd()),
		windows.LOCKFILE_FAIL_IMMEDIATELY|windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, len, 0, &windows.Overlapped{Offset: start}),
		IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len uint32) (bool, xErrorCode) {
	rc := vfsOS.readLock(file, start, len)
	if rc == _OK {
		vfsOS.unlock(file, start, len)
	}
	return rc != _OK, _OK
}

func (vfsOSMethods) lockErrorCode(err error, def xErrorCode) xErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(syscall.Errno); ok {
		// https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
		switch errno {
		case
			windows.ERROR_LOCK_VIOLATION,
			windows.ERROR_IO_PENDING:
			return xErrorCode(BUSY)
		}
	}
	return def
}
