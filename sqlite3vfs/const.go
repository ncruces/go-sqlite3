package sqlite3vfs

// OpenFlag is a flag for the [VFS.Open] method.
//
// https://www.sqlite.org/c3ref/c_open_autoproxy.html
type OpenFlag uint32

const (
	OPEN_READONLY      OpenFlag = 0x00000001 /* Ok for sqlite3_open_v2() */
	OPEN_READWRITE     OpenFlag = 0x00000002 /* Ok for sqlite3_open_v2() */
	OPEN_CREATE        OpenFlag = 0x00000004 /* Ok for sqlite3_open_v2() */
	OPEN_DELETEONCLOSE OpenFlag = 0x00000008 /* VFS only */
	OPEN_EXCLUSIVE     OpenFlag = 0x00000010 /* VFS only */
	OPEN_AUTOPROXY     OpenFlag = 0x00000020 /* VFS only */
	OPEN_URI           OpenFlag = 0x00000040 /* Ok for sqlite3_open_v2() */
	OPEN_MEMORY        OpenFlag = 0x00000080 /* Ok for sqlite3_open_v2() */
	OPEN_MAIN_DB       OpenFlag = 0x00000100 /* VFS only */
	OPEN_TEMP_DB       OpenFlag = 0x00000200 /* VFS only */
	OPEN_TRANSIENT_DB  OpenFlag = 0x00000400 /* VFS only */
	OPEN_MAIN_JOURNAL  OpenFlag = 0x00000800 /* VFS only */
	OPEN_TEMP_JOURNAL  OpenFlag = 0x00001000 /* VFS only */
	OPEN_SUBJOURNAL    OpenFlag = 0x00002000 /* VFS only */
	OPEN_SUPER_JOURNAL OpenFlag = 0x00004000 /* VFS only */
	OPEN_NOMUTEX       OpenFlag = 0x00008000 /* Ok for sqlite3_open_v2() */
	OPEN_FULLMUTEX     OpenFlag = 0x00010000 /* Ok for sqlite3_open_v2() */
	OPEN_SHAREDCACHE   OpenFlag = 0x00020000 /* Ok for sqlite3_open_v2() */
	OPEN_PRIVATECACHE  OpenFlag = 0x00040000 /* Ok for sqlite3_open_v2() */
	OPEN_WAL           OpenFlag = 0x00080000 /* VFS only */
	OPEN_NOFOLLOW      OpenFlag = 0x01000000 /* Ok for sqlite3_open_v2() */
)

// AccessFlag is a flag for the [VFS.Access] method.
//
// https://www.sqlite.org/c3ref/c_access_exists.html
type AccessFlag uint32

const (
	ACCESS_EXISTS    AccessFlag = 0
	ACCESS_READWRITE AccessFlag = 1 /* Used by PRAGMA temp_store_directory */
	ACCESS_READ      AccessFlag = 2 /* Unused */
)

// SyncFlag is a flag for the [File.Sync] method.
//
// https://www.sqlite.org/c3ref/c_sync_dataonly.html
type SyncFlag uint32

const (
	SYNC_NORMAL   SyncFlag = 0x00002
	SYNC_FULL     SyncFlag = 0x00003
	SYNC_DATAONLY SyncFlag = 0x00010
)

// LockLevel is a value used with [File.Lock] and [File.Unlock] methods.
//
// https://www.sqlite.org/c3ref/c_lock_exclusive.html
type LockLevel uint32

const (
	// No locks are held on the database.
	// The database may be neither read nor written.
	// Any internally cached data is considered suspect and subject to
	// verification against the database file before being used.
	// Other processes can read or write the database as their own locking
	// states permit.
	// This is the default state.
	LOCK_NONE LockLevel = 0 /* xUnlock() only */

	// The database may be read but not written.
	// Any number of processes can hold SHARED locks at the same time,
	// hence there can be many simultaneous readers.
	// But no other thread or process is allowed to write to the database file
	// while one or more SHARED locks are active.
	LOCK_SHARED LockLevel = 1 /* xLock() or xUnlock() */

	// A RESERVED lock means that the process is planning on writing to the
	// database file at some point in the future but that it is currently just
	// reading from the file.
	// Only a single RESERVED lock may be active at one time,
	// though multiple SHARED locks can coexist with a single RESERVED lock.
	// RESERVED differs from PENDING in that new SHARED locks can be acquired
	// while there is a RESERVED lock.
	LOCK_RESERVED LockLevel = 2 /* xLock() only */

	// A PENDING lock means that the process holding the lock wants to write to
	// the database as soon as possible and is just waiting on all current
	// SHARED locks to clear so that it can get an EXCLUSIVE lock.
	// No new SHARED locks are permitted against the database if a PENDING lock
	// is active, though existing SHARED locks are allowed to continue.
	LOCK_PENDING LockLevel = 3 /* internal use only */

	// An EXCLUSIVE lock is needed in order to write to the database file.
	// Only one EXCLUSIVE lock is allowed on the file and no other locks of any
	// kind are allowed to coexist with an EXCLUSIVE lock.
	// In order to maximize concurrency, SQLite works to minimize the amount of
	// time that EXCLUSIVE locks are held.
	LOCK_EXCLUSIVE LockLevel = 4 /* xLock() only */
)
