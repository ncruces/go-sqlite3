//go:build !windows && !linux && !darwin

package sqlite3

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	return file.Sync()
}

func (vfsOSMethods) Allocate(file *os.File, size int64) error {
	return notImplErr
}

func (vfsOSMethods) fcntlGetLock(file *os.File, lock *unix.Flock_t) error {
	return notImplErr
}

func (vfsOSMethods) fcntlSetLock(file *os.File, lock unix.Flock_t) error {
	return notImplErr
}

func (vfsOSMethods) fcntlSetLockTimeout(file *os.File, lock unix.Flock_t, timeout time.Duration) error {
	return notImplErr
}
