//go:build sqlite3_dotlk

package vfs

import (
	"os"
	"sync"
)

var (
	// +checklocks:vfsDotLocksMtx
	vfsDotLocks    = map[string]*vfsDotLocker{}
	vfsDotLocksMtx sync.Mutex
)

type vfsDotLocker struct {
	shared   int32 // +checklocks:vfsDotLocksMtx
	pending  bool  // +checklocks:vfsDotLocksMtx
	reserved bool  // +checklocks:vfsDotLocksMtx
}

func osGetSharedLock(file *os.File) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		f, err := os.OpenFile(name+".lock", os.O_CREATE|os.O_EXCL, os.ModeDir|0700)
		if err != nil {
			return _BUSY // Another process has the lock.
		}
		f.Close()

		locker = &vfsDotLocker{}
		vfsDotLocks[name] = locker
	}

	if locker.pending {
		return _BUSY
	}
	locker.shared++
	return _OK
}

func osGetReservedLock(file *os.File) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_LOCK
	}

	if locker.reserved {
		return _BUSY
	}
	locker.reserved = true
	return _OK
}

func osGetExclusiveLock(file *os.File, state *LockLevel) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_LOCK
	}

	if *state < LOCK_PENDING {
		locker.pending = true
		*state = LOCK_PENDING
	}
	if locker.shared > 1 {
		return _BUSY
	}
	return _OK
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_UNLOCK
	}

	if state >= LOCK_RESERVED {
		locker.reserved = false
	}
	if state >= LOCK_PENDING {
		locker.pending = false
	}
	return _OK
}

func osReleaseLock(file *os.File, state LockLevel) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_UNLOCK
	}

	if locker.shared == 1 {
		if err := os.Remove(name + ".lock"); err != nil {
			return _IOERR_UNLOCK
		}
		delete(vfsDotLocks, name)
	}

	if state >= LOCK_RESERVED {
		locker.reserved = false
	}
	if state >= LOCK_PENDING {
		locker.pending = false
	}
	locker.shared--
	return _OK
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return false, _OK
	}
	return locker.reserved, _OK
}
