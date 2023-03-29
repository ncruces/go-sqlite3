package vfs

import (
	"context"
	"os"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

const (
	// No locks are held on the database.
	// The database may be neither read nor written.
	// Any internally cached data is considered suspect and subject to
	// verification against the database file before being used.
	// Other processes can read or write the database as their own locking
	// states permit.
	// This is the default state.
	_NO_LOCK = 0

	// The database may be read but not written.
	// Any number of processes can hold SHARED locks at the same time,
	// hence there can be many simultaneous readers.
	// But no other thread or process is allowed to write to the database file
	// while one or more SHARED locks are active.
	_SHARED_LOCK = 1

	// A RESERVED lock means that the process is planning on writing to the
	// database file at some point in the future but that it is currently just
	// reading from the file.
	// Only a single RESERVED lock may be active at one time,
	// though multiple SHARED locks can coexist with a single RESERVED lock.
	// RESERVED differs from PENDING in that new SHARED locks can be acquired
	// while there is a RESERVED lock.
	_RESERVED_LOCK = 2

	// A PENDING lock means that the process holding the lock wants to write to
	// the database as soon as possible and is just waiting on all current
	// SHARED locks to clear so that it can get an EXCLUSIVE lock.
	// No new SHARED locks are permitted against the database if a PENDING lock
	// is active, though existing SHARED locks are allowed to continue.
	_PENDING_LOCK = 3

	// An EXCLUSIVE lock is needed in order to write to the database file.
	// Only one EXCLUSIVE lock is allowed on the file and no other locks of any
	// kind are allowed to coexist with an EXCLUSIVE lock.
	// In order to maximize concurrency, SQLite works to minimize the amount of
	// time that EXCLUSIVE locks are held.
	_EXCLUSIVE_LOCK = 4

	_PENDING_BYTE  = 0x40000000
	_RESERVED_BYTE = (_PENDING_BYTE + 1)
	_SHARED_FIRST  = (_PENDING_BYTE + 2)
	_SHARED_SIZE   = 510
)

type vfsLockState uint32

func vfsLock(ctx context.Context, mod api.Module, pFile uint32, eLock vfsLockState) _ErrorCode {
	// Argument check. SQLite never explicitly requests a pending lock.
	if eLock != _SHARED_LOCK && eLock != _RESERVED_LOCK && eLock != _EXCLUSIVE_LOCK {
		panic(util.AssertErr())
	}

	file := getOSFile(ctx, mod, pFile)
	cLock := getFileLock(ctx, mod, pFile)
	timeout := getFileLockTimeout(ctx, mod, pFile)
	readOnly := getFileReadOnly(ctx, mod, pFile)

	switch {
	case cLock < _NO_LOCK || cLock > _EXCLUSIVE_LOCK:
		// Connection state check.
		panic(util.AssertErr())
	case cLock == _NO_LOCK && eLock > _SHARED_LOCK:
		// We never move from unlocked to anything higher than a shared lock.
		panic(util.AssertErr())
	case cLock != _SHARED_LOCK && eLock == _RESERVED_LOCK:
		// A shared lock is always held when a reserved lock is requested.
		panic(util.AssertErr())
	}

	// If we already have an equal or more restrictive lock, do nothing.
	if cLock >= eLock {
		return _OK
	}

	// Do not allow any kind of write-lock on a read-only database.
	if readOnly && eLock > _RESERVED_LOCK {
		return _IOERR_LOCK
	}

	switch eLock {
	case _SHARED_LOCK:
		// Must be unlocked to get SHARED.
		if cLock != _NO_LOCK {
			panic(util.AssertErr())
		}
		if rc := osGetSharedLock(file, timeout); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _SHARED_LOCK)
		return _OK

	case _RESERVED_LOCK:
		// Must be SHARED to get RESERVED.
		if cLock != _SHARED_LOCK {
			panic(util.AssertErr())
		}
		if rc := osGetReservedLock(file, timeout); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _RESERVED_LOCK)
		return _OK

	case _EXCLUSIVE_LOCK:
		// Must be SHARED, RESERVED or PENDING to get EXCLUSIVE.
		if cLock <= _NO_LOCK || cLock >= _EXCLUSIVE_LOCK {
			panic(util.AssertErr())
		}
		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if cLock < _PENDING_LOCK {
			if rc := osGetPendingLock(file); rc != _OK {
				return rc
			}
			setFileLock(ctx, mod, pFile, _PENDING_LOCK)
		}
		if rc := osGetExclusiveLock(file, timeout); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _EXCLUSIVE_LOCK)
		return _OK

	default:
		panic(util.AssertErr())
	}
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile uint32, eLock vfsLockState) _ErrorCode {
	// Argument check.
	if eLock != _NO_LOCK && eLock != _SHARED_LOCK {
		panic(util.AssertErr())
	}

	file := getOSFile(ctx, mod, pFile)
	cLock := getFileLock(ctx, mod, pFile)

	// Connection state check.
	if cLock < _NO_LOCK || cLock > _EXCLUSIVE_LOCK {
		panic(util.AssertErr())
	}

	// If we don't have a more restrictive lock, do nothing.
	if cLock <= eLock {
		return _OK
	}

	switch eLock {
	case _SHARED_LOCK:
		if rc := osDowngradeLock(file, cLock); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _SHARED_LOCK)
		return _OK

	case _NO_LOCK:
		rc := osReleaseLock(file, cLock)
		setFileLock(ctx, mod, pFile, _NO_LOCK)
		return rc

	default:
		panic(util.AssertErr())
	}
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) _ErrorCode {
	file := getOSFile(ctx, mod, pFile)
	cLock := getFileLock(ctx, mod, pFile)

	// Connection state check.
	if cLock < _NO_LOCK || cLock > _EXCLUSIVE_LOCK {
		panic(util.AssertErr())
	}

	var locked bool
	var rc _ErrorCode
	if cLock >= _RESERVED_LOCK {
		locked = true
	} else {
		locked, rc = osCheckReservedLock(file)
	}

	var res uint32
	if locked {
		res = 1
	}
	util.WriteUint32(mod, pResOut, res)
	return rc
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
