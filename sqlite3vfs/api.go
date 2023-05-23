// Package sqlite3vfs wraps the C SQLite VFS API.
package sqlite3vfs

// A VFS defines the interface between the SQLite core and the underlying operating system.
//
// Use sqlite3.ErrorCode or sqlite3.ExtendedErrorCode to return specific error codes to SQLite.
//
// https://www.sqlite.org/c3ref/vfs.html
type VFS interface {
	Open(name string, flags OpenFlag) (File, OpenFlag, error)
	Delete(name string, syncDir bool) error
	Access(name string, flags AccessFlag) (bool, error)
	FullPathname(name string) (string, error)
}

// A File represents an open file in the OS interface layer.
//
// Use sqlite3.ErrorCode or sqlite3.ExtendedErrorCode to return specific error codes to SQLite.
// In particular, sqlite3.BUSY is necessary to correctly implement lock methods.
//
// https://www.sqlite.org/c3ref/io_methods.html
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

// FileLockState extends [File] to implement the
// SQLITE_FCNTL_LOCKSTATE file control opcode.
//
// https://www.sqlite.org/c3ref/c_fcntl_begin_atomic_write.html
type FileLockState interface {
	File
	LockState() LockLevel
}

// FileSizeHint extends [File] to implement the
// SQLITE_FCNTL_SIZE_HINT file control opcode.
//
// https://www.sqlite.org/c3ref/c_fcntl_begin_atomic_write.html
type FileSizeHint interface {
	File
	SizeHint(size int64) error
}

// FileHasMoved extends [File] to implement the
// SQLITE_FCNTL_HAS_MOVED file control opcode.
//
// https://www.sqlite.org/c3ref/c_fcntl_begin_atomic_write.html
type FileHasMoved interface {
	File
	HasMoved() (bool, error)
}

// FilePowersafeOverwrite extends [File] to implement the
// SQLITE_FCNTL_POWERSAFE_OVERWRITE file control opcode.
//
// https://www.sqlite.org/c3ref/c_fcntl_begin_atomic_write.html
type FilePowersafeOverwrite interface {
	File
	PowersafeOverwrite() bool
	SetPowersafeOverwrite(bool)
}