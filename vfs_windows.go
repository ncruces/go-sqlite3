package sqlite3

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func deleteOnClose(f *os.File) {}

func (l *vfsFileLocker) GetShared() xErrorCode {
	// Acquire the SHARED lock.
	return l.readLock(_SHARED_FIRST, _SHARED_SIZE)
}

func (l *vfsFileLocker) GetReserved() xErrorCode {
	// Acquire the RESERVED lock.
	return l.writeLock(_RESERVED_BYTE, 1)
}

func (l *vfsFileLocker) GetPending() xErrorCode {
	// Acquire the PENDING lock.
	return l.writeLock(_PENDING_BYTE, 1)
}

func (l *vfsFileLocker) GetExclusive() xErrorCode {
	// Release the SHARED lock.
	l.unlock(_SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	rc := l.writeLock(_SHARED_FIRST, _SHARED_SIZE)

	// Reacquire the SHARED lock.
	if rc != _OK {
		l.readLock(_SHARED_FIRST, _SHARED_SIZE)
	}
	return rc
}

func (l *vfsFileLocker) Downgrade() xErrorCode {
	// Release the SHARED lock.
	l.unlock(_SHARED_FIRST, _SHARED_SIZE)

	// Reacquire the SHARED lock.
	if rc := l.readLock(_SHARED_FIRST, _SHARED_SIZE); rc != _OK {
		// This should never happen.
		// We should always be able to reacquire the read lock.
		return IOERR_RDLOCK
	}

	// Release the PENDING and RESERVED locks.
	l.unlock(_RESERVED_BYTE, 1)
	l.unlock(_PENDING_BYTE, 1)
	return _OK
}

func (l *vfsFileLocker) Release() xErrorCode {
	// Release all locks.
	l.unlock(_SHARED_FIRST, _SHARED_SIZE)
	l.unlock(_RESERVED_BYTE, 1)
	l.unlock(_PENDING_BYTE, 1)
	return _OK
}

func (l *vfsFileLocker) CheckReserved() (bool, xErrorCode) {
	// Test the RESERVED lock.
	rc := l.readLock(_RESERVED_BYTE, 1)
	if rc == _OK {
		l.unlock(_RESERVED_BYTE, 1)
	}
	return rc != _OK, _OK
}

func (l *vfsFileLocker) CheckPending() (bool, xErrorCode) {
	// Test the PENDING lock.
	rc := l.readLock(_PENDING_BYTE, 1)
	if rc == _OK {
		l.unlock(_PENDING_BYTE, 1)
	}
	return rc != _OK, _OK
}

func (l *vfsFileLocker) unlock(start, len uint32) xErrorCode {
	err := windows.UnlockFileEx(windows.Handle(l.file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (l *vfsFileLocker) readLock(start, len uint32) xErrorCode {
	return l.errorCode(windows.LockFileEx(windows.Handle(l.file.Fd()),
		windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, len, 0, &windows.Overlapped{Offset: start}),
		IOERR_LOCK)
}

func (l *vfsFileLocker) writeLock(start, len uint32) xErrorCode {
	return l.errorCode(windows.LockFileEx(windows.Handle(l.file.Fd()),
		windows.LOCKFILE_FAIL_IMMEDIATELY|windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, len, 0, &windows.Overlapped{Offset: start}),
		IOERR_LOCK)
}

func (*vfsFileLocker) errorCode(err error, def xErrorCode) xErrorCode {
	if err == nil {
		return _OK
	}
	if errno, _ := err.(syscall.Errno); errno == windows.ERROR_INVALID_HANDLE {
		return def
	}
	return xErrorCode(BUSY)
}
