package sqlite3

import "os"

func deleteOnClose(f *os.File) {}

func (l *vfsFileLocker) GetShared() ExtendedErrorCode {
	return _OK
}

func (l *vfsFileLocker) GetReserved() ExtendedErrorCode {
	return _OK
}

func (l *vfsFileLocker) GetPending() ExtendedErrorCode {
	return _OK
}

func (l *vfsFileLocker) GetExclusive() ExtendedErrorCode {
	return _OK
}

func (l *vfsFileLocker) Downgrade() ExtendedErrorCode {
	return _OK
}

func (l *vfsFileLocker) Release() ExtendedErrorCode {
	return _OK
}

func (l *vfsFileLocker) CheckReserved() (bool, ExtendedErrorCode) {
	return false, _OK
}
