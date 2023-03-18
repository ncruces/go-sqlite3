package sqlite3

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	if dataonly {
		//lint:ignore SA1019 OK on linux
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

func (vfsOSMethods) fcntlGetLock(file *os.File, lock *unix.Flock_t) error {
	return unix.FcntlFlock(file.Fd(), unix.F_OFD_GETLK, lock)
}

func (vfsOSMethods) fcntlSetLock(file *os.File, lock unix.Flock_t) error {
	return unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
}

func (vfsOSMethods) fcntlSetLockTimeout(file *os.File, lock unix.Flock_t, timeout time.Duration) error {
	for {
		err := unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
		if errno, _ := err.(unix.Errno); errno != unix.EAGAIN {
			return err
		}
		if timeout < time.Millisecond {
			return err
		}
		timeout -= time.Millisecond
		time.Sleep(time.Millisecond)
	}
}
