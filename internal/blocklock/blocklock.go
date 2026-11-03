// Package blocklock serializes access to the blocks of a file
// that VFSes encrypt with length-preserving ciphers.
//
// These VFSes update partial blocks with a read-update-write,
// which changes the ciphertext of the entire block.
// A concurrent read of the same block, which the OS
// need not make atomic with the write, may return
// a mix of old and new ciphertext, which decrypts to garbage,
// even for the parts of the block that didn't change.
package blocklock

import (
	"sync"

	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

// Lock serializes access to the blocks of a file
// across connections.
// A nil Lock does nothing.
type Lock struct {
	fd   uintptr // file locks, if supported
	mtx  *rwMutex
	name string
}

// New returns a Lock for a file opened with name and flags,
// or nil if the file can't be accessed concurrently.
func New(file vfs.File, name *vfs.Filename, flags vfs.OpenFlag) *Lock {
	// Only main databases and WALs are shared between connections.
	if name == nil || flags&(vfs.OPEN_MAIN_DB|vfs.OPEN_WAL) == 0 {
		return nil
	}
	// Prefer file locks, which also work across processes.
	if f, ok := vfsutil.UnwrapFile[osFile](file); ok {
		if fd := f.Fd(); osSupported(fd) {
			return &Lock{fd: fd}
		}
	}
	// Otherwise, lock within the process.
	return &Lock{mtx: acquire(name.String()), name: name.String()}
}

type osFile interface {
	vfs.File
	Fd() uintptr
}

// Close releases resources associated with the Lock.
func (l *Lock) Close() {
	if l != nil && l.mtx != nil {
		release(l.name)
		l.mtx = nil
	}
}

// RLock acquires a shared lock on bytes [start, end) of the file.
func (l *Lock) RLock(start, end int64) error {
	switch {
	case l == nil:
		return nil
	case l.mtx != nil:
		l.mtx.RLock()
		return nil
	default:
		return osReadLock(l.fd, start, end)
	}
}

// RUnlock releases a shared lock acquired by RLock.
func (l *Lock) RUnlock(start, end int64) {
	switch {
	case l == nil:
	case l.mtx != nil:
		l.mtx.RUnlock()
	default:
		osUnlock(l.fd, start, end)
	}
}

// Lock acquires an exclusive lock on bytes [start, end) of the file.
func (l *Lock) Lock(start, end int64) error {
	switch {
	case l == nil:
		return nil
	case l.mtx != nil:
		l.mtx.Lock()
		return nil
	default:
		return osWriteLock(l.fd, start, end)
	}
}

// Unlock releases an exclusive lock acquired by Lock.
func (l *Lock) Unlock(start, end int64) {
	switch {
	case l == nil:
	case l.mtx != nil:
		l.mtx.Unlock()
	default:
		osUnlock(l.fd, start, end)
	}
}

type rwMutex struct {
	sync.RWMutex
	refs int // +checklocks:mutexesMtx
}

var (
	// +checklocks:mutexesMtx
	mutexes    = map[string]*rwMutex{}
	mutexesMtx sync.Mutex
)

func acquire(name string) *rwMutex {
	mutexesMtx.Lock()
	defer mutexesMtx.Unlock()

	m := mutexes[name]
	if m == nil {
		m = &rwMutex{}
		mutexes[name] = m
	}
	m.refs++
	return m
}

func release(name string) {
	mutexesMtx.Lock()
	defer mutexesMtx.Unlock()

	m := mutexes[name]
	if m.refs--; m.refs == 0 {
		delete(mutexes, name)
	}
}
