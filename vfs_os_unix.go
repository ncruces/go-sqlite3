//go:build unix

package sqlite3

import (
	"io/fs"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (vfsOSMethods) Access(path string, flags _AccessFlag) error {
	var access uint32 = unix.F_OK
	switch flags {
	case _ACCESS_READWRITE:
		access = unix.R_OK | unix.W_OK
	case _ACCESS_READ:
		access = unix.R_OK
	}
	return unix.Access(path, access)
}

func (vfsOSMethods) GetExclusiveLock(file *os.File, timeout time.Duration) xErrorCode {
	if timeout == 0 {
		timeout = time.Millisecond
	}

	// Acquire the EXCLUSIVE lock.
	return vfsOS.writeLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)
}

func (vfsOSMethods) DowngradeLock(file *os.File, state vfsLockState) xErrorCode {
	if state >= _EXCLUSIVE_LOCK {
		// Downgrade to a SHARED lock.
		if rc := vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
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
	err := vfsOS.fcntlSetLock(file, unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) readLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lockErrorCode(vfsOS.fcntlSetLockTimeout(file, unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}, timeout), IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	// TODO: implement timeouts.
	return vfsOS.lockErrorCode(vfsOS.fcntlSetLockTimeout(file, unix.Flock_t{
		Type:  unix.F_WRLCK,
		Start: start,
		Len:   len,
	}, timeout), IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len int64) (bool, xErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if vfsOS.fcntlGetLock(file, &lock) != nil {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != unix.F_UNLCK, _OK
}

func (vfsOSMethods) lockErrorCode(err error, def xErrorCode) xErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(unix.Errno); ok {
		switch errno {
		case
			unix.EACCES,
			unix.EAGAIN,
			unix.EBUSY,
			unix.EINTR,
			unix.ENOLCK,
			unix.EDEADLK,
			unix.ETIMEDOUT:
			return xErrorCode(BUSY)
		case unix.EPERM:
			return xErrorCode(PERM)
		}
	}
	return def
}
