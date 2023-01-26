package sqlite3

import "os"

func deleteOnClose(f *os.File) {}

func (l *vfsFileLocker) LockState() vfsLockState {
	return l.state
}

func (l *vfsFileLocker) LockShared() xErrorCode {
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsFileLocker) LockReserved() xErrorCode {
	l.state = _RESERVED_LOCK
	return _OK
}

func (l *vfsFileLocker) LockPending() xErrorCode {
	l.state = _PENDING_LOCK
	return _OK
}

func (l *vfsFileLocker) LockExclusive() xErrorCode {
	l.state = _EXCLUSIVE_LOCK
	return _OK
}

func (l *vfsFileLocker) DowngradeLock() xErrorCode {
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsFileLocker) Unlock() xErrorCode {
	l.state = _NO_LOCK
	return _OK
}

func (l *vfsFileLocker) CheckReservedLock() (bool, xErrorCode) {
	if l.state >= _RESERVED_LOCK {
		return true, _OK
	}
	return false, _OK
}
