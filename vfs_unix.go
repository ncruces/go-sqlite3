//go:build unix

package sqlite3

import (
	"os"
	"runtime"
	"syscall"
)

func (vfsOSMethods) DeleteOnClose(file *os.File) {
	_ = os.Remove(file.Name())
}

func (vfsOSMethods) GetExclusiveLock(file *os.File) xErrorCode {
	// Acquire the EXCLUSIVE lock.
	return vfsOS.writeLock(file, _SHARED_FIRST, _SHARED_SIZE)
}

func (vfsOSMethods) DowngradeLock(file *os.File, state vfsLockState) xErrorCode {
	if state >= _EXCLUSIVE_LOCK {
		// Downgrade to a SHARED lock.
		if rc := vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE); rc != _OK {
			// In theory, the downgrade to a SHARED cannot fail because another
			// process is holding an incompatible lock. If it does, this
			// indicates that the other process is not following the locking
			// protocol. If this happens, return IOERR_RDLOCK. Returning
			// BUSY would confuse the upper layer.
			return IOERR_RDLOCK
		}
	}
	// Release the PENDING and RESERVED locks.
	return vfsOS.unlock(file, _PENDING_BYTE, 2)
}

func (vfsOSMethods) ReleaseLock(file *os.File, _ vfsLockState) xErrorCode {
	// Release all locks.
	return vfsOS.unlock(file, 0, 0)
}

func (vfsOSMethods) unlock(file *os.File, start, len int64) xErrorCode {
	err := vfsOS.fcntlSetLock(file, &syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) readLock(file *os.File, start, len int64) xErrorCode {
	return vfsOS.lockErrorCode(vfsOS.fcntlSetLock(file, &syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: start,
		Len:   len,
	}), IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len int64) xErrorCode {
	return vfsOS.lockErrorCode(vfsOS.fcntlSetLock(file, &syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: start,
		Len:   len,
	}), IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len int64) (bool, xErrorCode) {
	lock := syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if vfsOS.fcntlGetLock(file, &lock) != nil {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != syscall.F_UNLCK, _OK
}

func (vfsOSMethods) fcntlGetLock(file *os.File, lock *syscall.Flock_t) error {
	var F_OFD_GETLK int
	switch runtime.GOOS {
	case "linux":
		// https://github.com/torvalds/linux/blob/master/include/uapi/asm-generic/fcntl.h
		F_OFD_GETLK = 36 // F_OFD_GETLK
	case "darwin":
		// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
		F_OFD_GETLK = 92 // F_OFD_GETLK
	case "illumos":
		// https://github.com/illumos/illumos-gate/blob/master/usr/src/uts/common/sys/fcntl.h
		F_OFD_GETLK = 47 // F_OFD_GETLK
	default:
		return notImplErr
	}
	return syscall.FcntlFlock(file.Fd(), F_OFD_GETLK, lock)
}

func (vfsOSMethods) fcntlSetLock(file *os.File, lock *syscall.Flock_t) error {
	var F_OFD_SETLK int
	switch runtime.GOOS {
	case "linux":
		// https://github.com/torvalds/linux/blob/master/include/uapi/asm-generic/fcntl.h
		F_OFD_SETLK = 37 // F_OFD_SETLK
	case "darwin":
		// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
		F_OFD_SETLK = 90 // F_OFD_SETLK
	case "illumos":
		// https://github.com/illumos/illumos-gate/blob/master/usr/src/uts/common/sys/fcntl.h
		F_OFD_SETLK = 48 // F_OFD_SETLK
	default:
		return notImplErr
	}
	return syscall.FcntlFlock(file.Fd(), F_OFD_SETLK, lock)
}

func (vfsOSMethods) lockErrorCode(err error, def xErrorCode) xErrorCode {
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
