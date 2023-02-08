package sqlite3

import "os"

func deleteOnClose(f *os.File) {}

func (l *vfsFileLocker) GetShared() xErrorCode {
	return _OK
}

func (l *vfsFileLocker) GetReserved() xErrorCode {
	return _OK
}

func (l *vfsFileLocker) GetPending() xErrorCode {
	return _OK
}

func (l *vfsFileLocker) GetExclusive() xErrorCode {
	return _OK
}

func (l *vfsFileLocker) Downgrade() xErrorCode {
	return _OK
}

func (l *vfsFileLocker) Release() xErrorCode {
	return _OK
}

func (l *vfsFileLocker) CheckReserved() (bool, xErrorCode) {
	return false, _OK
}
