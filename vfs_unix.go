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
	// Acquire the EXCLUSIVE lock.
	return l.writeLock(_SHARED_FIRST, _SHARED_SIZE)
}

func (l *vfsFileLocker) Downgrade() xErrorCode {
	// Downgrade to a SHARED lock.
	if rc := l.readLock(_SHARED_FIRST, _SHARED_SIZE); rc != _OK {
		// In theory, the downgrade to a SHARED cannot fail because another
		// process is holding an incompatible lock. If it does, this
		// indicates that the other process is not following the locking
		// protocol. If this happens, return IOERR_RDLOCK. Returning
		// BUSY would confuse the upper layer.
		return IOERR_RDLOCK
	}

	// Release the PENDING and RESERVED locks.
	return l.unlock(_PENDING_BYTE, 2)
}

func (l *vfsFileLocker) Release() xErrorCode {
	// Release all locks.
	return l.unlock(0, 0)
}

func (l *vfsFileLocker) CheckReserved() (bool, xErrorCode) {
	// Test the RESERVED lock.
	return l.checkLock(_RESERVED_BYTE, 1)
}

func (l *vfsFileLocker) CheckPending() (bool, xErrorCode) {
	// Test the PENDING lock.
	return l.checkLock(_PENDING_BYTE, 1)
}

func (l *vfsFileLocker) unlock(start, len int64) xErrorCode {
	err := l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (l *vfsFileLocker) readLock(start, len int64) xErrorCode {
	return l.errorCode(l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: start,
		Len:   len,
	}), IOERR_LOCK)
}

func (l *vfsFileLocker) writeLock(start, len int64) xErrorCode {
	return l.errorCode(l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: start,
		Len:   len,
	}), IOERR_LOCK)
}

func (l *vfsFileLocker) checkLock(start, len int64) (bool, xErrorCode) {
	lock := syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: start,
		Len:   len,
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
	case "illumos":
		// https://github.com/illumos/illumos-gate/blob/master/usr/src/uts/common/sys/fcntl.h
		F_GETLK = 47 // F_OFD_GETLK
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
	case "illumos":
		// https://github.com/illumos/illumos-gate/blob/master/usr/src/uts/common/sys/fcntl.h
		F_SETLK = 48 // F_OFD_SETLK
	}
	return syscall.FcntlFlock(l.file.Fd(), F_SETLK, lock)
}

func (*vfsFileLocker) errorCode(err error, def xErrorCode) xErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case
			syscall.EACCES,
			syscall.EAGAIN,
			syscall.EBUSY,
			syscall.EINTR,
			syscall.ENOLCK,
			syscall.EDEADLK,
			syscall.ETIMEDOUT:
			return xErrorCode(BUSY)
		case syscall.EPERM:
			return xErrorCode(PERM)
		}
	}
	return def
}
