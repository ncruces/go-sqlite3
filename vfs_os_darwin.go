package sqlite3

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	if !fullsync {
		return unix.Fsync(int(file.Fd()))
	}
	return file.Sync()
}

func (vfsOSMethods) Allocate(file *os.File, size int64) error {
	// https://stackoverflow.com/a/11497568/867786
	store := unix.Fstore_t{
		Flags:   unix.F_ALLOCATECONTIG,
		Posmode: unix.F_PEOFPOSMODE,
		Offset:  0,
		Length:  size,
	}

	// Try to get a continous chunk of disk space.
	err := unix.FcntlFstore(file.Fd(), unix.F_PREALLOCATE, &store)
	if err != nil {
		// OK, perhaps we are too fragmented, allocate non-continuous.
		store.Flags = unix.F_ALLOCATEALL
		return unix.FcntlFstore(file.Fd(), unix.F_PREALLOCATE, &store)
	}
	return nil
}

func (vfsOSMethods) fcntlGetLock(file *os.File, lock *unix.Flock_t) error {
	const F_OFD_GETLK = 92 // https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
	return unix.FcntlFlock(file.Fd(), F_OFD_GETLK, lock)
}

func (vfsOSMethods) fcntlSetLock(file *os.File, lock unix.Flock_t) error {
	const F_OFD_SETLK = 90 // https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
	return unix.FcntlFlock(file.Fd(), F_OFD_SETLK, &lock)
}

func (vfsOSMethods) fcntlSetLockTimeout(timeout time.Duration, file *os.File, lock unix.Flock_t) error {
	if timeout == 0 {
		return vfsOS.fcntlSetLock(file, lock)
	}

	const F_OFD_SETLKWTIMEOUT = 93 // https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
	flocktimeout := &struct {
		unix.Flock_t
		unix.Timespec
	}{
		Flock_t:  lock,
		Timespec: unix.NsecToTimespec(int64(timeout / time.Nanosecond)),
	}
	return unix.FcntlFlock(file.Fd(), F_OFD_SETLKWTIMEOUT, &flocktimeout.Flock_t)
}
