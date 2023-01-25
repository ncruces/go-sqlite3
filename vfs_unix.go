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

type vfsFileLocker struct {
	*os.File
	state vfsLockState
}

func (l *vfsFileLocker) LockState() vfsLockState {
	return l.state
}

func (l *vfsFileLocker) LockShared() xErrorCode {
	// A PENDING lock is needed before acquiring a SHARED lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _PENDING_BYTE,
		Len:   1,
	}) {
		return IOERR_LOCK
	}

	// Acquire the SHARED lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _SHARED_FIRST,
		Len:   _SHARED_SIZE,
	}) {
		return IOERR_LOCK
	}
	l.state = _SHARED_LOCK

	// Relese the PENDING lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: _PENDING_BYTE,
		Len:   1,
	}) {
		return IOERR_UNLOCK
	}

	return _OK
}

func (l *vfsFileLocker) LockReserved() xErrorCode {
	// Acquire the RESERVED lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: _RESERVED_BYTE,
		Len:   1,
	}) {
		return IOERR_LOCK
	}
	l.state = _RESERVED_LOCK
	return _OK
}

func (l *vfsFileLocker) LockPending() xErrorCode {
	// Acquire the PENDING lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: _PENDING_BYTE,
		Len:   1,
	}) {
		return IOERR_LOCK
	}
	l.state = _PENDING_LOCK
	return _OK
}

func (l *vfsFileLocker) LockExclusive() xErrorCode {
	// Acquire the EXCLUSIVE lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: _SHARED_FIRST,
		Len:   _SHARED_SIZE,
	}) {
		return IOERR_LOCK
	}
	l.state = _EXCLUSIVE_LOCK
	return _OK
}

func (l *vfsFileLocker) DowngradeLock() xErrorCode {
	// Downgrade to a SHARED lock.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_RDLCK,
		Start: _SHARED_FIRST,
		Len:   _SHARED_SIZE,
	}) {
		return IOERR_RDLOCK
	}
	l.state = _SHARED_LOCK

	// Release the PENDING and RESERVED locks.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: _PENDING_BYTE,
		Len:   2,
	}) {
		return IOERR_UNLOCK
	}
	return _OK
}

func (l *vfsFileLocker) Unlock() xErrorCode {
	// Release all locks.
	if !l.fcntlSetLock(&syscall.Flock_t{
		Type: syscall.F_UNLCK,
	}) {
		return IOERR_UNLOCK
	}
	l.state = _NO_LOCK
	return _OK
}

func (l *vfsFileLocker) CheckReservedLock() (bool, xErrorCode) {
	if l.state >= _RESERVED_LOCK {
		return true, _OK
	}
	// Test all write locks.
	lock := syscall.Flock_t{
		Type: syscall.F_RDLCK,
	}
	if !l.fcntlGetLock(&lock) {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type == syscall.F_UNLCK, _OK
}

func (l *vfsFileLocker) fcntlGetLock(lock *syscall.Flock_t) bool {
	F_GETLK := syscall.F_GETLK
	if runtime.GOOS == "linux" {
		F_GETLK = 36 // F_OFD_GETLK
	}
	return syscall.FcntlFlock(l.Fd(), F_GETLK, lock) == nil
}

func (l *vfsFileLocker) fcntlSetLock(lock *syscall.Flock_t) bool {
	F_SETLK := syscall.F_SETLK
	if runtime.GOOS == "linux" {
		F_SETLK = 37 // F_OFD_SETLK
	}
	return syscall.FcntlFlock(l.Fd(), F_SETLK, lock) == nil
}
