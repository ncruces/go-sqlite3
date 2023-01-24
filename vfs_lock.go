package sqlite3

import (
	"context"

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

type vfsLocker interface {
	LockState() uint32
	LockShared() uint32    // UNLOCKED -> SHARED
	LockReserved() uint32  // SHARED -> RESERVED
	LockPending() uint32   // SHARED|RESERVED -> (PENDING)
	LockExclusive() uint32 // PENDING -> EXCLUSIVE
	DowngradeLock() uint32 // SHARED <- EXCLUSIVE|PENDING|RESERVED
	ReleaseLock() uint32   // UNLOCKED <- EXCLUSIVE|PENDING|RESERVED|SHARED
}

func vfsLock(ctx context.Context, mod api.Module, pFile, eLock uint32) uint32 {
	if assert && (eLock == _NO_LOCK || eLock == _PENDING_LOCK) {
		panic(assertErr + " [d4oxww]")
	}

	ptr := vfsFilePtr{mod, pFile}
	cLock := ptr.Lock()

	// If we already have an equal or more restrictive lock, do nothing.
	if cLock >= eLock {
		return _OK
	}

	if assert {
		switch {
		case cLock == _NO_LOCK && eLock > _SHARED_LOCK:
			// We never move from unlocked to anything higher than shared lock.
			panic(assertErr + " [pfa77m]")
		case cLock != _SHARED_LOCK && eLock == _RESERVED_LOCK:
			// A shared lock is always held when a reserve lock is requested.
			panic(assertErr + " [5cfmsp]")
		}
	}

	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()
	of := vfsOpenFiles[ptr.ID()]
	fLock := of.LockState()

	// If some other connection has a lock that precludes the requested lock, return BUSY.
	if cLock != fLock && (eLock > _SHARED_LOCK || fLock >= _PENDING_LOCK) {
		return uint32(BUSY)
	}
	if eLock == _EXCLUSIVE_LOCK && of.shared > 1 {
		// We are trying for an exclusive lock but another connection in this
		// same process is still holding a shared lock.
		return uint32(BUSY)
	}

	// If a SHARED lock is requested, and some other connection has a SHARED or RESERVED lock,
	// then increment the reference count and return OK.
	if eLock == _SHARED_LOCK && (fLock == _SHARED_LOCK || fLock == _RESERVED_LOCK) {
		if assert && !(cLock == _NO_LOCK && of.shared > 0) {
			panic(assertErr + " [k7coz6]")
		}
		ptr.SetLock(_SHARED_LOCK)
		of.shared++
		return _OK
	}

	// If control gets to this point, then actually go ahead and make
	// operating system calls for the specified lock.
	switch eLock {
	case _SHARED_LOCK:
		if assert && !(fLock == _NO_LOCK && of.shared == 0) {
			panic(assertErr + " [jsyttq]")
		}
		if rc := of.LockShared(); rc != _OK {
			return uint32(rc)
		}
		of.shared = 1
		ptr.SetLock(_SHARED_LOCK)
		return _OK

	case _RESERVED_LOCK:
		if rc := of.LockReserved(); rc != _OK {
			return uint32(rc)
		}
		ptr.SetLock(_RESERVED_LOCK)
		return _OK

	case _EXCLUSIVE_LOCK:
		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if cLock < _PENDING_LOCK {
			if rc := of.LockPending(); rc != _OK {
				return uint32(rc)
			}
			ptr.SetLock(_PENDING_LOCK)
		}
		if rc := of.LockExclusive(); rc != _OK {
			return uint32(rc)
		}
		ptr.SetLock(_EXCLUSIVE_LOCK)
		return _OK

	default:
		panic(assertErr + " [56ng2l]")
	}
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile, eLock uint32) (rc uint32) {
	if assert && (eLock != _NO_LOCK && eLock != _SHARED_LOCK) {
		panic(assertErr + " [7i4jw3]")
	}

	ptr := vfsFilePtr{mod, pFile}
	cLock := ptr.Lock()

	// If we don't have a more restrictive lock, do nothing.
	if cLock <= eLock {
		return _OK
	}

	vfsOpenFilesMtx.Lock()
	defer vfsOpenFilesMtx.Unlock()
	of := vfsOpenFiles[ptr.ID()]
	fLock := of.LockState()

	if assert && of.shared <= 0 {
		panic(assertErr + " [2bhkwg]")
	}
	if cLock > _SHARED_LOCK {
		if assert && cLock != fLock {
			panic(assertErr + " [6pmjqf]")
		}
		if eLock == _SHARED_LOCK {
			if rc := of.DowngradeLock(); rc != _OK {
				// In theory, the downgrade to a SHARED cannot fail because another
				// process is holding an incompatible lock. If it does, this
				// indicates that the other process is not following the locking
				// protocol. If this happens, return IOERR_RDLOCK. Returning
				// BUSY would confuse the upper layer (in practice it causes
				// an assert to fail).
				return uint32(IOERR_RDLOCK)
			}
			ptr.SetLock(_SHARED_LOCK)
			return _OK
		}
	}

	if assert && eLock != _NO_LOCK {
		panic(assertErr + " [gilo9p]")
	}
	// Decrement the shared lock counter. Release the file lock
	// only when all connections have released the lock.
	switch {
	case of.shared > 1:
		ptr.SetLock(_NO_LOCK)
		of.shared--
		return _OK

	case of.shared == 1:
		if rc := of.ReleaseLock(); rc != _OK {
			return uint32(IOERR_UNLOCK)
		}
		ptr.SetLock(_NO_LOCK)
		of.shared = 0
		return _OK

	default:
		panic(assertErr + " [50gw51]")
	}
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) uint32 {
	mod.Memory().WriteUint32Le(pResOut, 0)
	return _OK
}
