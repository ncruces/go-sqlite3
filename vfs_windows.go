package sqlite3

import "os"

func deleteOnClose(f *os.File) {}

func (l *vfsFileLocker) GetShared() ExtendedErrorCode {
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsFileLocker) GetReserved() ExtendedErrorCode {
	l.state = _RESERVED_LOCK
	return _OK
}

func (l *vfsFileLocker) GetPending() ExtendedErrorCode {
	l.state = _PENDING_LOCK
	return _OK
}

func (l *vfsFileLocker) GetExclusive() ExtendedErrorCode {
	l.state = _EXCLUSIVE_LOCK
	return _OK
}

func (l *vfsFileLocker) Downgrade() ExtendedErrorCode {
	l.state = _SHARED_LOCK
	return _OK
}

func (l *vfsFileLocker) Release() ExtendedErrorCode {
	l.state = _NO_LOCK
	return _OK
}

func (l *vfsFileLocker) CheckReserved() (bool, ExtendedErrorCode) {
	if l.state >= _RESERVED_LOCK {
		return true, _OK
	}
	return false, _OK
}
