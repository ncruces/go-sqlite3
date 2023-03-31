package vfs

import (
	"context"
	"os"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

const (
	_PENDING_BYTE  = 0x40000000
	_RESERVED_BYTE = (_PENDING_BYTE + 1)
	_SHARED_FIRST  = (_PENDING_BYTE + 2)
	_SHARED_SIZE   = 510
)

func vfsLock(ctx context.Context, mod api.Module, pFile uint32, eLock _LockLevel) _ErrorCode {
	// Argument check. SQLite never explicitly requests a pending lock.
	if eLock != _LOCK_SHARED && eLock != _LOCK_RESERVED && eLock != _LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	file := getOSFile(ctx, mod, pFile)
	cLock := getFileLock(ctx, mod, pFile)
	timeout := getFileLockTimeout(ctx, mod, pFile)
	readOnly := getFileReadOnly(ctx, mod, pFile)

	switch {
	case cLock < _LOCK_NONE || cLock > _LOCK_EXCLUSIVE:
		// Connection state check.
		panic(util.AssertErr())
	case cLock == _LOCK_NONE && eLock > _LOCK_SHARED:
		// We never move from unlocked to anything higher than a shared lock.
		panic(util.AssertErr())
	case cLock != _LOCK_SHARED && eLock == _LOCK_RESERVED:
		// A shared lock is always held when a reserved lock is requested.
		panic(util.AssertErr())
	}

	// If we already have an equal or more restrictive lock, do nothing.
	if cLock >= eLock {
		return _OK
	}

	// Do not allow any kind of write-lock on a read-only database.
	if readOnly && eLock > _LOCK_RESERVED {
		return _IOERR_LOCK
	}

	switch eLock {
	case _LOCK_SHARED:
		// Must be unlocked to get SHARED.
		if cLock != _LOCK_NONE {
			panic(util.AssertErr())
		}
		if rc := osGetSharedLock(file, timeout); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _LOCK_SHARED)
		return _OK

	case _LOCK_RESERVED:
		// Must be SHARED to get RESERVED.
		if cLock != _LOCK_SHARED {
			panic(util.AssertErr())
		}
		if rc := osGetReservedLock(file, timeout); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _LOCK_RESERVED)
		return _OK

	case _LOCK_EXCLUSIVE:
		// Must be SHARED, RESERVED or PENDING to get EXCLUSIVE.
		if cLock <= _LOCK_NONE || cLock >= _LOCK_EXCLUSIVE {
			panic(util.AssertErr())
		}
		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if cLock < _LOCK_PENDING {
			if rc := osGetPendingLock(file); rc != _OK {
				return rc
			}
			setFileLock(ctx, mod, pFile, _LOCK_PENDING)
		}
		if rc := osGetExclusiveLock(file, timeout); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _LOCK_EXCLUSIVE)
		return _OK

	default:
		panic(util.AssertErr())
	}
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile uint32, eLock _LockLevel) _ErrorCode {
	// Argument check.
	if eLock != _LOCK_NONE && eLock != _LOCK_SHARED {
		panic(util.AssertErr())
	}

	file := getOSFile(ctx, mod, pFile)
	cLock := getFileLock(ctx, mod, pFile)

	// Connection state check.
	if cLock < _LOCK_NONE || cLock > _LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	// If we don't have a more restrictive lock, do nothing.
	if cLock <= eLock {
		return _OK
	}

	switch eLock {
	case _LOCK_SHARED:
		if rc := osDowngradeLock(file, cLock); rc != _OK {
			return rc
		}
		setFileLock(ctx, mod, pFile, _LOCK_SHARED)
		return _OK

	case _LOCK_NONE:
		rc := osReleaseLock(file, cLock)
		setFileLock(ctx, mod, pFile, _LOCK_NONE)
		return rc

	default:
		panic(util.AssertErr())
	}
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) _ErrorCode {
	file := getOSFile(ctx, mod, pFile)
	cLock := getFileLock(ctx, mod, pFile)

	// Connection state check.
	if cLock < _LOCK_NONE || cLock > _LOCK_EXCLUSIVE {
		panic(util.AssertErr())
	}

	var locked bool
	var rc _ErrorCode
	if cLock >= _LOCK_RESERVED {
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
