package sqlite3

import (
	"context"
	"os"
	"sync"

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

type vfsFileLocker struct {
	sync.Mutex
	file   *os.File
	state  vfsLockState
	shared int
}

func vfsLock(ctx context.Context, mod api.Module, pFile uint32, eLock vfsLockState) uint32 {
	// SQLite never explicitly requests a pendig lock.
	if eLock != _SHARED_LOCK && eLock != _RESERVED_LOCK && eLock != _EXCLUSIVE_LOCK {
		panic(assertErr())
	}

	ptr := vfsFilePtr{mod, pFile}
	cLock := ptr.Lock()

	// If we already have an equal or more restrictive lock, do nothing.
	if cLock >= eLock {
		return _OK
	}

	switch {
	case cLock == _NO_LOCK && eLock > _SHARED_LOCK:
		// We never move from unlocked to anything higher than a shared lock.
		panic(assertErr())
	case cLock != _SHARED_LOCK && eLock == _RESERVED_LOCK:
		// A shared lock is always held when a reserved lock is requested.
		panic(assertErr())
	}

	fLock := ptr.Locker()
	fLock.Lock()
	defer fLock.Unlock()

	// If some other connection has a lock that precludes the requested lock, return BUSY.
	if cLock != fLock.state && (eLock > _SHARED_LOCK || fLock.state >= _PENDING_LOCK) {
		return uint32(BUSY)
	}

	// If a SHARED lock is requested, and some other connection has a SHARED or RESERVED lock,
	// then increment the reference count and return OK.
	if eLock == _SHARED_LOCK && (fLock.state == _SHARED_LOCK || fLock.state == _RESERVED_LOCK) {
		if cLock != _NO_LOCK || fLock.shared <= 0 {
			panic(assertErr())
		}
		ptr.SetLock(_SHARED_LOCK)
		fLock.shared++
		return _OK
	}

	// If control gets to this point, then actually go ahead and make
	// operating system calls for the specified lock.
	switch eLock {
	case _SHARED_LOCK:
		if fLock.state != _NO_LOCK || fLock.shared != 0 {
			panic(assertErr())
		}
		if rc := fLock.GetShared(); rc != _OK {
			return uint32(rc)
		}
		ptr.SetLock(_SHARED_LOCK)
		fLock.state = _SHARED_LOCK
		fLock.shared = 1
		return _OK

	case _RESERVED_LOCK:
		if fLock.state != _SHARED_LOCK || fLock.shared <= 0 {
			panic(assertErr())
		}
		if rc := fLock.GetReserved(); rc != _OK {
			return uint32(rc)
		}
		ptr.SetLock(_RESERVED_LOCK)
		fLock.state = _RESERVED_LOCK
		return _OK

	case _EXCLUSIVE_LOCK:
		if fLock.state <= _NO_LOCK || fLock.state >= _EXCLUSIVE_LOCK || fLock.shared <= 0 {
			panic(assertErr())
		}

		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if fLock.state == _RESERVED_LOCK {
			if rc := fLock.GetPending(); rc != _OK {
				return uint32(rc)
			}
			ptr.SetLock(_PENDING_LOCK)
			fLock.state = _PENDING_LOCK
		}

		// We are trying for an EXCLUSIVE lock but another connection is still holding a shared lock.
		if fLock.shared > 1 {
			return uint32(BUSY)
		}

		if rc := fLock.GetExclusive(); rc != _OK {
			return uint32(rc)
		}
		ptr.SetLock(_EXCLUSIVE_LOCK)
		fLock.state = _EXCLUSIVE_LOCK
		return _OK

	default:
		panic(assertErr())
	}
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile uint32, eLock vfsLockState) uint32 {
	if eLock != _NO_LOCK && eLock != _SHARED_LOCK {
		panic(assertErr())
	}

	ptr := vfsFilePtr{mod, pFile}
	cLock := ptr.Lock()

	// If we don't have a more restrictive lock, do nothing.
	if cLock <= eLock {
		return _OK
	}

	fLock := ptr.Locker()
	fLock.Lock()
	defer fLock.Unlock()

	if fLock.shared <= 0 {
		panic(assertErr())
	}
	if cLock > _SHARED_LOCK {
		if cLock != fLock.state {
			panic(assertErr())
		}
		if eLock == _SHARED_LOCK {
			if rc := fLock.Downgrade(); rc != _OK {
				return uint32(rc)
			}
			ptr.SetLock(_SHARED_LOCK)
			fLock.state = _SHARED_LOCK
			return _OK
		}
	}

	if eLock != _NO_LOCK {
		panic(assertErr())
	}

	// Release the connection lock and decrement the shared lock counter.
	// Release the file lock only when all connections have released the lock.
	ptr.SetLock(_NO_LOCK)
	if fLock.shared--; fLock.shared == 0 {
		fLock.state = _NO_LOCK
		return uint32(fLock.Release())
	}
	return _OK
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) uint32 {
	ptr := vfsFilePtr{mod, pFile}
	cLock := ptr.Lock()

	if cLock > _SHARED_LOCK {
		panic(assertErr())
	}

	fLock := ptr.Locker()
	fLock.Lock()
	defer fLock.Unlock()

	if fLock.state >= _RESERVED_LOCK {
		memory{mod}.writeUint32(pResOut, 1)
		return _OK
	}

	locked, rc := fLock.CheckReserved()
	var res uint32
	if locked {
		res = 1
	}
	memory{mod}.writeUint32(pResOut, res)
	return uint32(rc)
}
