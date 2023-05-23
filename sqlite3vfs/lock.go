package sqlite3vfs

import (
	"os"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

const (
	_PENDING_BYTE  = 0x40000000
	_RESERVED_BYTE = (_PENDING_BYTE + 1)
	_SHARED_FIRST  = (_PENDING_BYTE + 2)
	_SHARED_SIZE   = 510
)

func (file *vfsFile) Lock(eLock LockLevel) error {
	// Argument check. SQLite never explicitly requests a pending lock.
	if eLock != LOCK_SHARED && eLock != LOCK_RESERVED && eLock != LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	switch {
	case file.lock < LOCK_NONE || file.lock > LOCK_EXCLUSIVE:
		// Connection state check.
		panic(util.AssertErr())
	case file.lock == LOCK_NONE && eLock > LOCK_SHARED:
		// We never move from unlocked to anything higher than a shared lock.
		panic(util.AssertErr())
	case file.lock != LOCK_SHARED && eLock == LOCK_RESERVED:
		// A shared lock is always held when a reserved lock is requested.
		panic(util.AssertErr())
	}

	// If we already have an equal or more restrictive lock, do nothing.
	if file.lock >= eLock {
		return nil
	}

	// Do not allow any kind of write-lock on a read-only database.
	if file.readOnly && eLock >= LOCK_RESERVED {
		return _IOERR_LOCK
	}

	switch eLock {
	case LOCK_SHARED:
		// Must be unlocked to get SHARED.
		if file.lock != LOCK_NONE {
			panic(util.AssertErr())
		}
		if rc := osGetSharedLock(file.File, file.lockTimeout); rc != _OK {
			return rc
		}
		file.lock = LOCK_SHARED
		return nil

	case LOCK_RESERVED:
		// Must be SHARED to get RESERVED.
		if file.lock != LOCK_SHARED {
			panic(util.AssertErr())
		}
		if rc := osGetReservedLock(file.File, file.lockTimeout); rc != _OK {
			return rc
		}
		file.lock = LOCK_RESERVED
		return nil

	case LOCK_EXCLUSIVE:
		// Must be SHARED, RESERVED or PENDING to get EXCLUSIVE.
		if file.lock <= LOCK_NONE || file.lock >= LOCK_EXCLUSIVE {
			panic(util.AssertErr())
		}
		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if file.lock < LOCK_PENDING {
			if rc := osGetPendingLock(file.File); rc != _OK {
				return rc
			}
			file.lock = LOCK_PENDING
		}
		if rc := osGetExclusiveLock(file.File, file.lockTimeout); rc != _OK {
			return rc
		}
		file.lock = LOCK_EXCLUSIVE
		return nil

	default:
		panic(util.AssertErr())
	}
}

func (file *vfsFile) Unlock(eLock LockLevel) error {
	// Argument check.
	if eLock != LOCK_NONE && eLock != LOCK_SHARED {
		panic(util.AssertErr())
	}

	// Connection state check.
	if file.lock < LOCK_NONE || file.lock > LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	// If we don't have a more restrictive lock, do nothing.
	if file.lock <= eLock {
		return nil
	}

	switch eLock {
	case LOCK_SHARED:
		if rc := osDowngradeLock(file.File, file.lock); rc != _OK {
			return rc
		}
		file.lock = LOCK_SHARED
		return nil

	case LOCK_NONE:
		rc := osReleaseLock(file.File, file.lock)
		file.lock = LOCK_NONE
		return rc

	default:
		panic(util.AssertErr())
	}
}

func (file *vfsFile) CheckReservedLock() (bool, error) {
	// Connection state check.
	if file.lock < LOCK_NONE || file.lock > LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	if file.lock >= LOCK_RESERVED {
		return true, nil
	}
	return osCheckReservedLock(file.File)
}

func osGetReservedLock(file *os.File, timeout time.Duration) _ErrorCode {
	// Acquire the RESERVED lock.
	return osWriteLock(file, _RESERVED_BYTE, 1, timeout)
}

func osGetPendingLock(file *os.File) _ErrorCode {
	// Acquire the PENDING lock.
	return osWriteLock(file, _PENDING_BYTE, 1, 0)
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	// Test the RESERVED lock.
	return osCheckLock(file, _RESERVED_BYTE, 1)
}
