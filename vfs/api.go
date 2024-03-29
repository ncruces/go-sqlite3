// Package vfs wraps the C SQLite VFS API.
package vfs

import "net/url"

// A VFS defines the interface between the SQLite core and the underlying operating system.
//
// Use sqlite3.ErrorCode or sqlite3.ExtendedErrorCode to return specific error codes to SQLite.
//
// https://sqlite.org/c3ref/vfs.html
type VFS interface {
	Open(name string, flags OpenFlag) (File, OpenFlag, error)
	Delete(name string, syncDir bool) error
	Access(name string, flags AccessFlag) (bool, error)
	FullPathname(name string) (string, error)
}

// VFSParams extends VFS with the ability to handle URI parameters
// through the OpenParams method.
//
// https://sqlite.org/c3ref/uri_boolean.html
type VFSParams interface {
	VFS
	OpenParams(name string, flags OpenFlag, params url.Values) (File, OpenFlag, error)
}

// A File represents an open file in the OS interface layer.
//
// Use sqlite3.ErrorCode or sqlite3.ExtendedErrorCode to return specific error codes to SQLite.
// In particular, sqlite3.BUSY is necessary to correctly implement lock methods.
//
// https://sqlite.org/c3ref/io_methods.html
type File interface {
	Close() error
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Truncate(size int64) error
	Sync(flags SyncFlag) error
	Size() (int64, error)
	Lock(lock LockLevel) error
	Unlock(lock LockLevel) error
	CheckReservedLock() (bool, error)
	SectorSize() int
	DeviceCharacteristics() DeviceCharacteristic
}

// FileLockState extends File to implement the
// SQLITE_FCNTL_LOCKSTATE file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntllockstate
type FileLockState interface {
	File
	LockState() LockLevel
}

// FileSizeHint extends File to implement the
// SQLITE_FCNTL_SIZE_HINT file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntlsizehint
type FileSizeHint interface {
	File
	SizeHint(size int64) error
}

// FileHasMoved extends File to implement the
// SQLITE_FCNTL_HAS_MOVED file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntlhasmoved
type FileHasMoved interface {
	File
	HasMoved() (bool, error)
}

// FileOverwrite extends File to implement the
// SQLITE_FCNTL_OVERWRITE file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntloverwrite
type FileOverwrite interface {
	File
	Overwrite() error
}

// FilePersistentWAL extends File to implement the
// SQLITE_FCNTL_PERSIST_WAL file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntlpersistwal
type FilePersistentWAL interface {
	File
	PersistentWAL() bool
	SetPersistentWAL(bool)
}

// FilePowersafeOverwrite extends File to implement the
// SQLITE_FCNTL_POWERSAFE_OVERWRITE file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntlpowersafeoverwrite
type FilePowersafeOverwrite interface {
	File
	PowersafeOverwrite() bool
	SetPowersafeOverwrite(bool)
}

// FileCommitPhaseTwo extends File to implement the
// SQLITE_FCNTL_COMMIT_PHASETWO file control opcode.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntlcommitphasetwo
type FileCommitPhaseTwo interface {
	File
	CommitPhaseTwo() error
}

// FileBatchAtomicWrite extends File to implement the
// SQLITE_FCNTL_BEGIN_ATOMIC_WRITE, SQLITE_FCNTL_COMMIT_ATOMIC_WRITE
// and SQLITE_FCNTL_ROLLBACK_ATOMIC_WRITE file control opcodes.
//
// https://sqlite.org/c3ref/c_fcntl_begin_atomic_write.html#sqlitefcntlbeginatomicwrite
type FileBatchAtomicWrite interface {
	File
	BeginAtomicWrite() error
	CommitAtomicWrite() error
	RollbackAtomicWrite() error
}
