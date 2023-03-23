package sqlite3

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	if dataonly {
		_, _, err := unix.Syscall(unix.SYS_FDATASYNC, file.Fd(), 0, 0)
		if err != 0 {
			return err
		}
		return nil
	}
	return file.Sync()
}

func (vfsOSMethods) Allocate(file *os.File, size int64) error {
	if size == 0 {
		return nil
	}
	return unix.Fallocate(int(file.Fd()), 0, 0, size)
}

func (vfsOSMethods) unlock(file *os.File, start, len int64) xErrorCode {
	err := unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) lock(file *os.File, typ int16, start, len int64, timeout time.Duration, def xErrorCode) xErrorCode {
	lock := unix.Flock_t{
		Type:  typ,
		Start: start,
		Len:   len,
	}
	var err error
	for {
		err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
		if errno, _ := err.(unix.Errno); errno != unix.EAGAIN {
			break
		}
		if timeout < time.Millisecond {
			break
		}
		timeout -= time.Millisecond
		time.Sleep(time.Millisecond)
	}
	return vfsOS.lockErrorCode(err, def)
}

func (vfsOSMethods) readLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lock(file, unix.F_RDLCK, start, len, timeout, IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lock(file, unix.F_WRLCK, start, len, timeout, IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len int64) (bool, xErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if unix.FcntlFlock(file.Fd(), unix.F_OFD_GETLK, &lock) != nil {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != unix.F_UNLCK, _OK
}
