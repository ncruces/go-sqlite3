//go:build unix

package sqlite3

import (
	"os"
	"runtime"
	"syscall"
)

func deleteOnClose(f *os.File) {
	_ = os.Remove(f.Name())
}

func (l *vfsFileLocker) GetShared() xErrorCode {
	// A PENDING lock is needed before acquiring a SHARED lock.
	if err := l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _PENDING_BYTE,
		Len:   1,
	}); err != nil {
		return l.errorCode(err, IOERR_LOCK)
	}

	// Acquire the SHARED lock.
	rc := l.errorCode(l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _SHARED_FIRST,
		Len:   _SHARED_SIZE,
	}), IOERR_LOCK)

	// Drop the temporary PENDING lock.
	if err := l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: _PENDING_BYTE,
		Len:   1,
	}); rc == _OK && err != nil {
		return IOERR_UNLOCK
	}
	return rc
}

func (l *vfsFileLocker) GetReserved() xErrorCode {
	// Acquire the RESERVED lock.
	return l.errorCode(l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: _RESERVED_BYTE,
		Len:   1,
	}), IOERR_LOCK)
}

func (l *vfsFileLocker) GetPending() xErrorCode {
	// Acquire the PENDING lock.
	return l.errorCode(l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: _PENDING_BYTE,
		Len:   1,
	}), IOERR_LOCK)
}

func (l *vfsFileLocker) GetExclusive() xErrorCode {
	// Acquire the EXCLUSIVE lock.
	return l.errorCode(l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: _SHARED_FIRST,
		Len:   _SHARED_SIZE,
	}), IOERR_LOCK)
}

func (l *vfsFileLocker) Downgrade() xErrorCode {
	// Downgrade to a SHARED lock.
	if err := l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _SHARED_FIRST,
		Len:   _SHARED_SIZE,
	}); err != nil {
		// In theory, the downgrade to a SHARED cannot fail because another
		// process is holding an incompatible lock. If it does, this
		// indicates that the other process is not following the locking
		// protocol. If this happens, return IOERR_RDLOCK. Returning
		// BUSY would confuse the upper layer.
		return IOERR_RDLOCK
	}

	// Release the PENDING and RESERVED locks.
	if err := l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: _PENDING_BYTE,
		Len:   2,
	}); err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (l *vfsFileLocker) Release() xErrorCode {
	// Release all locks.
	if err := l.fcntlSetLock(&syscall.Flock_t{
		Type: syscall.F_UNLCK,
	}); err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (l *vfsFileLocker) CheckReserved() (bool, xErrorCode) {
	// Test the RESERVED lock.
	lock := syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _RESERVED_BYTE,
		Len:   1,
	}
	if l.fcntlGetLock(&lock) != nil {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != syscall.F_UNLCK, _OK
}

func (l *vfsFileLocker) fcntlGetLock(lock *syscall.Flock_t) error {
	F_GETLK := syscall.F_GETLK
	switch runtime.GOOS {
	case "linux":
		// https://github.com/torvalds/linux/blob/master/include/uapi/asm-generic/fcntl.h
		F_GETLK = 36 // F_OFD_GETLK
	case "darwin":
		// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
		F_GETLK = 92 // F_OFD_GETLK
	}
	return syscall.FcntlFlock(l.file.Fd(), F_GETLK, lock)
}

func (l *vfsFileLocker) fcntlSetLock(lock *syscall.Flock_t) error {
	F_SETLK := syscall.F_SETLK
	switch runtime.GOOS {
	case "linux":
		// https://github.com/torvalds/linux/blob/master/include/uapi/asm-generic/fcntl.h
		F_SETLK = 37 // F_OFD_SETLK
	case "darwin":
		// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
		F_SETLK = 90 // F_OFD_SETLK
	}
	return syscall.FcntlFlock(l.file.Fd(), F_SETLK, lock)
}

func (*vfsFileLocker) errorCode(err error, def xErrorCode) xErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.EACCES:
		case syscall.EAGAIN:
		case syscall.EBUSY:
		case syscall.EINTR:
		case syscall.ENOLCK:
		case syscall.EDEADLK:
		case syscall.ETIMEDOUT:
			return xErrorCode(BUSY)
		case syscall.EPERM:
			return xErrorCode(PERM)
		}
	}
	return def
}
