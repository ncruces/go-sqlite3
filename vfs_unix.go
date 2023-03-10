//go:build unix

package sqlite3

import (
	"io/fs"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (vfsOSMethods) Access(path string, flags _AccessFlag) (bool, xErrorCode) {
	var access uint32 = unix.F_OK
	switch flags {
	case _ACCESS_READWRITE:
		access = unix.R_OK | unix.W_OK
	case _ACCESS_READ:
		access = unix.R_OK
	}

	err := unix.Access(path, access)
	if err == nil {
		return true, _OK
	}
	return false, _OK
}

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	if runtime.GOOS == "darwin" && !fullsync {
		return unix.Fsync(int(file.Fd()))
	}
	if runtime.GOOS == "linux" && dataonly {
		//lint:ignore SA1019 OK on linux
		_, _, err := unix.Syscall(unix.SYS_FDATASYNC, file.Fd(), 0, 0)
		if err != 0 {
			return err
		}
		return nil
	}
	return file.Sync()
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
	err := vfsOS.fcntlSetLock(file, &unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) readLock(file *os.File, start, len int64) xErrorCode {
	return vfsOS.lockErrorCode(vfsOS.fcntlSetLock(file, &unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}), IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len int64) xErrorCode {
	return vfsOS.lockErrorCode(vfsOS.fcntlSetLock(file, &unix.Flock_t{
		Type:  unix.F_WRLCK,
		Start: start,
		Len:   len,
	}), IOERR_LOCK)
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

func (vfsOSMethods) fcntlGetLock(file *os.File, lock *unix.Flock_t) error {
	var F_OFD_GETLK int
	switch runtime.GOOS {
	case "linux":
		// https://github.com/torvalds/linux/blob/master/include/uapi/asm-generic/fcntl.h
		F_OFD_GETLK = 36
	case "darwin":
		// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
		F_OFD_GETLK = 92
	case "illumos":
		// https://github.com/illumos/illumos-gate/blob/master/usr/src/uts/common/sys/fcntl.h
		F_OFD_GETLK = 47
	default:
		return notImplErr
	}
	return unix.FcntlFlock(file.Fd(), F_OFD_GETLK, lock)
}

func (vfsOSMethods) fcntlSetLock(file *os.File, lock *unix.Flock_t) error {
	var F_OFD_SETLK int
	switch runtime.GOOS {
	case "linux":
		// https://github.com/torvalds/linux/blob/master/include/uapi/asm-generic/fcntl.h
		F_OFD_SETLK = 37
	case "darwin":
		// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
		F_OFD_SETLK = 90
	case "illumos":
		// https://github.com/illumos/illumos-gate/blob/master/usr/src/uts/common/sys/fcntl.h
		F_OFD_SETLK = 48
	default:
		return notImplErr
	}
	return unix.FcntlFlock(file.Fd(), F_OFD_SETLK, lock)
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
