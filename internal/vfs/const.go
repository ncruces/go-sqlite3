package vfs

const (
	_MAX_PATHNAME        = 512
	_DEFAULT_SECTOR_SIZE = 4096
)

// https://www.sqlite.org/rescode.html
type _ErrorCode uint32

const (
	_OK       _ErrorCode = 0  /* Successful result */
	_PERM     _ErrorCode = 3  /* Access permission denied */
	_BUSY     _ErrorCode = 5  /* The database file is locked */
	_IOERR    _ErrorCode = 10 /* Some kind of disk I/O error occurred */
	_NOTFOUND _ErrorCode = 12 /* Unknown opcode in sqlite3_file_control() */
	_CANTOPEN _ErrorCode = 14 /* Unable to open the database file */

	_IOERR_READ              = _IOERR | (1 << 8)
	_IOERR_SHORT_READ        = _IOERR | (2 << 8)
	_IOERR_WRITE             = _IOERR | (3 << 8)
	_IOERR_FSYNC             = _IOERR | (4 << 8)
	_IOERR_DIR_FSYNC         = _IOERR | (5 << 8)
	_IOERR_TRUNCATE          = _IOERR | (6 << 8)
	_IOERR_FSTAT             = _IOERR | (7 << 8)
	_IOERR_UNLOCK            = _IOERR | (8 << 8)
	_IOERR_RDLOCK            = _IOERR | (9 << 8)
	_IOERR_DELETE            = _IOERR | (10 << 8)
	_IOERR_BLOCKED           = _IOERR | (11 << 8)
	_IOERR_NOMEM             = _IOERR | (12 << 8)
	_IOERR_ACCESS            = _IOERR | (13 << 8)
	_IOERR_CHECKRESERVEDLOCK = _IOERR | (14 << 8)
	_IOERR_LOCK              = _IOERR | (15 << 8)
	_IOERR_CLOSE             = _IOERR | (16 << 8)
	_IOERR_DIR_CLOSE         = _IOERR | (17 << 8)
	_IOERR_SHMOPEN           = _IOERR | (18 << 8)
	_IOERR_SHMSIZE           = _IOERR | (19 << 8)
	_IOERR_SHMLOCK           = _IOERR | (20 << 8)
	_IOERR_SHMMAP            = _IOERR | (21 << 8)
	_IOERR_SEEK              = _IOERR | (22 << 8)
	_IOERR_DELETE_NOENT      = _IOERR | (23 << 8)
	_IOERR_MMAP              = _IOERR | (24 << 8)
	_IOERR_GETTEMPPATH       = _IOERR | (25 << 8)
	_IOERR_CONVPATH          = _IOERR | (26 << 8)
	_IOERR_VNODE             = _IOERR | (27 << 8)
	_IOERR_AUTH              = _IOERR | (28 << 8)
	_IOERR_BEGIN_ATOMIC      = _IOERR | (29 << 8)
	_IOERR_COMMIT_ATOMIC     = _IOERR | (30 << 8)
	_IOERR_ROLLBACK_ATOMIC   = _IOERR | (31 << 8)
	_IOERR_DATA              = _IOERR | (32 << 8)
	_IOERR_CORRUPTFS         = _IOERR | (33 << 8)
	_CANTOPEN_NOTEMPDIR      = _CANTOPEN | (1 << 8)
	_CANTOPEN_ISDIR          = _CANTOPEN | (2 << 8)
	_CANTOPEN_FULLPATH       = _CANTOPEN | (3 << 8)
	_CANTOPEN_CONVPATH       = _CANTOPEN | (4 << 8)
	_CANTOPEN_DIRTYWAL       = _CANTOPEN | (5 << 8) /* Not Used */
	_CANTOPEN_SYMLINK        = _CANTOPEN | (6 << 8)
	_OK_SYMLINK              = _OK | (2 << 8) /* internal use only */
)

// https://www.sqlite.org/c3ref/c_open_autoproxy.html
type _OpenFlag uint32

const (
	_OPEN_READONLY      _OpenFlag = 0x00000001 /* Ok for sqlite3_open_v2() */
	_OPEN_READWRITE     _OpenFlag = 0x00000002 /* Ok for sqlite3_open_v2() */
	_OPEN_CREATE        _OpenFlag = 0x00000004 /* Ok for sqlite3_open_v2() */
	_OPEN_DELETEONCLOSE _OpenFlag = 0x00000008 /* VFS only */
	_OPEN_EXCLUSIVE     _OpenFlag = 0x00000010 /* VFS only */
	_OPEN_AUTOPROXY     _OpenFlag = 0x00000020 /* VFS only */
	_OPEN_URI           _OpenFlag = 0x00000040 /* Ok for sqlite3_open_v2() */
	_OPEN_MEMORY        _OpenFlag = 0x00000080 /* Ok for sqlite3_open_v2() */
	_OPEN_MAIN_DB       _OpenFlag = 0x00000100 /* VFS only */
	_OPEN_TEMP_DB       _OpenFlag = 0x00000200 /* VFS only */
	_OPEN_TRANSIENT_DB  _OpenFlag = 0x00000400 /* VFS only */
	_OPEN_MAIN_JOURNAL  _OpenFlag = 0x00000800 /* VFS only */
	_OPEN_TEMP_JOURNAL  _OpenFlag = 0x00001000 /* VFS only */
	_OPEN_SUBJOURNAL    _OpenFlag = 0x00002000 /* VFS only */
	_OPEN_SUPER_JOURNAL _OpenFlag = 0x00004000 /* VFS only */
	_OPEN_NOMUTEX       _OpenFlag = 0x00008000 /* Ok for sqlite3_open_v2() */
	_OPEN_FULLMUTEX     _OpenFlag = 0x00010000 /* Ok for sqlite3_open_v2() */
	_OPEN_SHAREDCACHE   _OpenFlag = 0x00020000 /* Ok for sqlite3_open_v2() */
	_OPEN_PRIVATECACHE  _OpenFlag = 0x00040000 /* Ok for sqlite3_open_v2() */
	_OPEN_WAL           _OpenFlag = 0x00080000 /* VFS only */
	_OPEN_NOFOLLOW      _OpenFlag = 0x01000000 /* Ok for sqlite3_open_v2() */
	_OPEN_EXRESCODE     _OpenFlag = 0x02000000 /* Extended result codes */
)

// https://www.sqlite.org/c3ref/c_access_exists.html
type _AccessFlag uint32

const (
	_ACCESS_EXISTS    _AccessFlag = 0
	_ACCESS_READWRITE _AccessFlag = 1 /* Used by PRAGMA temp_store_directory */
	_ACCESS_READ      _AccessFlag = 2 /* Unused */
)

// https://www.sqlite.org/c3ref/c_sync_dataonly.html
type _SyncFlag uint32

const (
	_SYNC_NORMAL   _SyncFlag = 0x00002
	_SYNC_FULL     _SyncFlag = 0x00003
	_SYNC_DATAONLY _SyncFlag = 0x00010
)

// https://www.sqlite.org/c3ref/c_lock_exclusive.html
type _LockLevel uint32

const (
	// No locks are held on the database.
	// The database may be neither read nor written.
	// Any internally cached data is considered suspect and subject to
	// verification against the database file before being used.
	// Other processes can read or write the database as their own locking
	// states permit.
	// This is the default state.
	_LOCK_NONE _LockLevel = 0 /* xUnlock() only */

	// The database may be read but not written.
	// Any number of processes can hold SHARED locks at the same time,
	// hence there can be many simultaneous readers.
	// But no other thread or process is allowed to write to the database file
	// while one or more SHARED locks are active.
	_LOCK_SHARED _LockLevel = 1 /* xLock() or xUnlock() */

	// A RESERVED lock means that the process is planning on writing to the
	// database file at some point in the future but that it is currently just
	// reading from the file.
	// Only a single RESERVED lock may be active at one time,
	// though multiple SHARED locks can coexist with a single RESERVED lock.
	// RESERVED differs from PENDING in that new SHARED locks can be acquired
	// while there is a RESERVED lock.
	_LOCK_RESERVED _LockLevel = 2 /* xLock() only */

	// A PENDING lock means that the process holding the lock wants to write to
	// the database as soon as possible and is just waiting on all current
	// SHARED locks to clear so that it can get an EXCLUSIVE lock.
	// No new SHARED locks are permitted against the database if a PENDING lock
	// is active, though existing SHARED locks are allowed to continue.
	_LOCK_PENDING _LockLevel = 3 /* internal use only */

	// An EXCLUSIVE lock is needed in order to write to the database file.
	// Only one EXCLUSIVE lock is allowed on the file and no other locks of any
	// kind are allowed to coexist with an EXCLUSIVE lock.
	// In order to maximize concurrency, SQLite works to minimize the amount of
	// time that EXCLUSIVE locks are held.
	_LOCK_EXCLUSIVE _LockLevel = 4 /* xLock() only */
)

// https://www.sqlite.org/c3ref/c_fcntl_begin_atomic_write.html
type _FcntlOpcode uint32

const (
	_FCNTL_LOCKSTATE             _FcntlOpcode = 1
	_FCNTL_GET_LOCKPROXYFILE     _FcntlOpcode = 2
	_FCNTL_SET_LOCKPROXYFILE     _FcntlOpcode = 3
	_FCNTL_LAST_ERRNO            _FcntlOpcode = 4
	_FCNTL_SIZE_HINT             _FcntlOpcode = 5
	_FCNTL_CHUNK_SIZE            _FcntlOpcode = 6
	_FCNTL_FILE_POINTER          _FcntlOpcode = 7
	_FCNTL_SYNC_OMITTED          _FcntlOpcode = 8
	_FCNTL_WIN32_AV_RETRY        _FcntlOpcode = 9
	_FCNTL_PERSIST_WAL           _FcntlOpcode = 10
	_FCNTL_OVERWRITE             _FcntlOpcode = 11
	_FCNTL_VFSNAME               _FcntlOpcode = 12
	_FCNTL_POWERSAFE_OVERWRITE   _FcntlOpcode = 13
	_FCNTL_PRAGMA                _FcntlOpcode = 14
	_FCNTL_BUSYHANDLER           _FcntlOpcode = 15
	_FCNTL_TEMPFILENAME          _FcntlOpcode = 16
	_FCNTL_MMAP_SIZE             _FcntlOpcode = 18
	_FCNTL_TRACE                 _FcntlOpcode = 19
	_FCNTL_HAS_MOVED             _FcntlOpcode = 20
	_FCNTL_SYNC                  _FcntlOpcode = 21
	_FCNTL_COMMIT_PHASETWO       _FcntlOpcode = 22
	_FCNTL_WIN32_SET_HANDLE      _FcntlOpcode = 23
	_FCNTL_WAL_BLOCK             _FcntlOpcode = 24
	_FCNTL_ZIPVFS                _FcntlOpcode = 25
	_FCNTL_RBU                   _FcntlOpcode = 26
	_FCNTL_VFS_POINTER           _FcntlOpcode = 27
	_FCNTL_JOURNAL_POINTER       _FcntlOpcode = 28
	_FCNTL_WIN32_GET_HANDLE      _FcntlOpcode = 29
	_FCNTL_PDB                   _FcntlOpcode = 30
	_FCNTL_BEGIN_ATOMIC_WRITE    _FcntlOpcode = 31
	_FCNTL_COMMIT_ATOMIC_WRITE   _FcntlOpcode = 32
	_FCNTL_ROLLBACK_ATOMIC_WRITE _FcntlOpcode = 33
	_FCNTL_LOCK_TIMEOUT          _FcntlOpcode = 34
	_FCNTL_DATA_VERSION          _FcntlOpcode = 35
	_FCNTL_SIZE_LIMIT            _FcntlOpcode = 36
	_FCNTL_CKPT_DONE             _FcntlOpcode = 37
	_FCNTL_RESERVE_BYTES         _FcntlOpcode = 38
	_FCNTL_CKPT_START            _FcntlOpcode = 39
	_FCNTL_EXTERNAL_READER       _FcntlOpcode = 40
	_FCNTL_CKSM_FILE             _FcntlOpcode = 41
	_FCNTL_RESET_CACHE           _FcntlOpcode = 42
)

// https://www.sqlite.org/c3ref/c_iocap_atomic.html
type _DeviceCharacteristic uint32

const (
	_IOCAP_ATOMIC                _DeviceCharacteristic = 0x00000001
	_IOCAP_ATOMIC512             _DeviceCharacteristic = 0x00000002
	_IOCAP_ATOMIC1K              _DeviceCharacteristic = 0x00000004
	_IOCAP_ATOMIC2K              _DeviceCharacteristic = 0x00000008
	_IOCAP_ATOMIC4K              _DeviceCharacteristic = 0x00000010
	_IOCAP_ATOMIC8K              _DeviceCharacteristic = 0x00000020
	_IOCAP_ATOMIC16K             _DeviceCharacteristic = 0x00000040
	_IOCAP_ATOMIC32K             _DeviceCharacteristic = 0x00000080
	_IOCAP_ATOMIC64K             _DeviceCharacteristic = 0x00000100
	_IOCAP_SAFE_APPEND           _DeviceCharacteristic = 0x00000200
	_IOCAP_SEQUENTIAL            _DeviceCharacteristic = 0x00000400
	_IOCAP_UNDELETABLE_WHEN_OPEN _DeviceCharacteristic = 0x00000800
	_IOCAP_POWERSAFE_OVERWRITE   _DeviceCharacteristic = 0x00001000
	_IOCAP_IMMUTABLE             _DeviceCharacteristic = 0x00002000
	_IOCAP_BATCH_ATOMIC          _DeviceCharacteristic = 0x00004000
)
