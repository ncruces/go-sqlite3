//go:build sqlite3_dotlk

package vfs

import (
	"bytes"
	"sync"
	"unsafe"
)

const (
	_WALINDEX_HDR_SIZE = 136
	_WALINDEX_PGSZ     = 32768
)

// The wal-index is kept in sync by copying at lock boundaries:
// acquire copies shared→private after a lock is taken,
// release copies private→shared before an exclusive lock is dropped,
// and a barrier does both. A per-connection shadow of the shared memory
// lets both directions copy only the words that actually changed.
//
// https://sqlite.org/walformat.html#the_wal_index_file_format
//
// Correctness requires two properties that plain word-copy loops do not
// provide on their own:
//
//  1. Copy operations of different connections must not interleave.
//     They are triggered by locks on *different* wal-index lock bytes
//     (e.g. a writer releasing WAL_WRITE_LOCK while a reader acquires a
//     READ_LOCK), so the file locks do not order them. An acquire that
//     overlaps a release can capture a fresh header together with
//     hole-ridden hash tables: SQLite then trusts hash lookups that are
//     missing frames, and a checkpointer acting on such a view copies
//     stale page versions into the database and truncates frames that
//     were never backfilled. A per-file lock (shmCopyLocks, keyed by the
//     -shm path) serializes all copies for that file; connections to
//     different files share no wal-index state and need no ordering.
//     (Native SQLite needs no such lock only because all connections read
//     and write one coherent mapping; its winShmNode similarly arbitrates
//     same-process access through shared state.)
//
//  2. A "nothing changed" fast path must compare exactly. The header
//     (first 136 bytes) is NOT a proxy for the whole page: SQLite
//     modifies hash-table words while leaving the header byte-identical —
//     e.g. a rolled-back transaction zeroes its hash entries via
//     walCleanupHash and restores the header — so a header-only check
//     skips real changes, the private and shared copies diverge
//     permanently, and after a checkpoint restart reuses frame numbers
//     the stale entries alias wrong frames. The skip below compares the
//     full page instead; bytes.Equal costs ~1µs per 32K page, which the
//     shadow-diff design already tolerates.
type shmCopyLock struct {
	sync.Mutex
	refs int // +checklocks:shmCopyLocksMtx
}

var (
	// +checklocks:shmCopyLocksMtx
	shmCopyLocks    = map[string]*shmCopyLock{}
	shmCopyLocksMtx sync.Mutex
)

func shmCopyLockGet(path string) *shmCopyLock {
	shmCopyLocksMtx.Lock()
	defer shmCopyLocksMtx.Unlock()
	l := shmCopyLocks[path]
	if l == nil {
		l = &shmCopyLock{}
		shmCopyLocks[path] = l
	}
	l.refs++
	return l
}

func shmCopyLockPut(path string) {
	shmCopyLocksMtx.Lock()
	defer shmCopyLocksMtx.Unlock()
	if l := shmCopyLocks[path]; l != nil {
		if l.refs--; l.refs <= 0 {
			delete(shmCopyLocks, path)
		}
	}
}

func (s *vfsShm) shmAcquire(errp *error) {
	if errp != nil && *errp != nil {
		return
	}
	if s.copyMu == nil {
		return // no copy state yet (no pages mapped)
	}
	s.copyMu.Lock()
	defer s.copyMu.Unlock()
	s.shmAcquireLocked()
}

func (s *vfsShm) shmRelease() {
	if s.copyMu == nil {
		return // no copy state yet (no pages mapped)
	}
	s.copyMu.Lock()
	defer s.copyMu.Unlock()
	s.shmReleaseLocked()
}

// shmAcquireLocked copies modified words from shared to private memory.
// Callers must hold the file's copy lock.
func (s *vfsShm) shmAcquireLocked() {
	for id, p := range s.ptrs {
		if bytes.Equal(s.shadow[id][:], s.shared[id][:]) {
			continue // page unchanged since this connection last synced
		}
		shared := shmPage(s.shared[id][:])
		shadow := shmPage(s.shadow[id][:])
		privat := shmPage(s.wrp.Bytes(p, _WALINDEX_PGSZ))
		for i, shared := range shared {
			if shadow[i] != shared {
				shadow[i] = shared
				privat[i] = shared
			}
		}
	}
}

// shmReleaseLocked copies modified words from private to shared memory.
// Callers must hold the file's copy lock.
func (s *vfsShm) shmReleaseLocked() {
	for id, p := range s.ptrs {
		if bytes.Equal(s.shadow[id][:], s.wrp.Bytes(p, _WALINDEX_PGSZ)) {
			continue // this connection made no changes to this page
		}
		shared := shmPage(s.shared[id][:])
		shadow := shmPage(s.shadow[id][:])
		privat := shmPage(s.wrp.Bytes(p, _WALINDEX_PGSZ))
		for i, privat := range privat {
			if shadow[i] != privat {
				shadow[i] = privat
				shared[i] = privat
			}
		}
	}
}

//go:nosplit
func shmPage(s []byte) *[_WALINDEX_PGSZ / 4]uint32 {
	p := (*uint32)(unsafe.Pointer(unsafe.SliceData(s)))
	return (*[_WALINDEX_PGSZ / 4]uint32)(unsafe.Slice(p, _WALINDEX_PGSZ/4))
}
