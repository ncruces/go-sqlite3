package vfs

import "github.com/ncruces/go-sqlite3/sqlite3vfs"

const (
	_MAX_STRING          = 512 // Used for short strings: names, error messages…
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
type _OpenFlag = sqlite3vfs.OpenFlag

const (
	_OPEN_READONLY      = sqlite3vfs.OPEN_READONLY
	_OPEN_READWRITE     = sqlite3vfs.OPEN_READWRITE
	_OPEN_CREATE        = sqlite3vfs.OPEN_CREATE
	_OPEN_DELETEONCLOSE = sqlite3vfs.OPEN_DELETEONCLOSE
	_OPEN_EXCLUSIVE     = sqlite3vfs.OPEN_EXCLUSIVE
	_OPEN_AUTOPROXY     = sqlite3vfs.OPEN_AUTOPROXY
	_OPEN_URI           = sqlite3vfs.OPEN_URI
	_OPEN_MEMORY        = sqlite3vfs.OPEN_MEMORY
	_OPEN_MAIN_DB       = sqlite3vfs.OPEN_MAIN_DB
	_OPEN_TEMP_DB       = sqlite3vfs.OPEN_TEMP_DB
	_OPEN_TRANSIENT_DB  = sqlite3vfs.OPEN_TRANSIENT_DB
	_OPEN_MAIN_JOURNAL  = sqlite3vfs.OPEN_MAIN_JOURNAL
	_OPEN_TEMP_JOURNAL  = sqlite3vfs.OPEN_TEMP_JOURNAL
	_OPEN_SUBJOURNAL    = sqlite3vfs.OPEN_SUBJOURNAL
	_OPEN_SUPER_JOURNAL = sqlite3vfs.OPEN_SUPER_JOURNAL
	_OPEN_NOMUTEX       = sqlite3vfs.OPEN_NOMUTEX
	_OPEN_FULLMUTEX     = sqlite3vfs.OPEN_FULLMUTEX
	_OPEN_SHAREDCACHE   = sqlite3vfs.OPEN_SHAREDCACHE
	_OPEN_PRIVATECACHE  = sqlite3vfs.OPEN_PRIVATECACHE
	_OPEN_WAL           = sqlite3vfs.OPEN_WAL
	_OPEN_NOFOLLOW      = sqlite3vfs.OPEN_NOFOLLOW
)

// https://www.sqlite.org/c3ref/c_access_exists.html
type _AccessFlag = sqlite3vfs.AccessFlag

const (
	_ACCESS_EXISTS    = sqlite3vfs.ACCESS_EXISTS
	_ACCESS_READWRITE = sqlite3vfs.ACCESS_READWRITE
	_ACCESS_READ      = sqlite3vfs.ACCESS_READ
)

// https://www.sqlite.org/c3ref/c_sync_dataonly.html
type _SyncFlag = sqlite3vfs.SyncFlag

const (
	_SYNC_NORMAL   = sqlite3vfs.SYNC_NORMAL
	_SYNC_FULL     = sqlite3vfs.SYNC_FULL
	_SYNC_DATAONLY = sqlite3vfs.SYNC_DATAONLY
)

// https://www.sqlite.org/c3ref/c_lock_exclusive.html
type _LockLevel = sqlite3vfs.LockLevel

const (
	_LOCK_NONE      = sqlite3vfs.LOCK_NONE
	_LOCK_SHARED    = sqlite3vfs.LOCK_SHARED
	_LOCK_RESERVED  = sqlite3vfs.LOCK_RESERVED
	_LOCK_PENDING   = sqlite3vfs.LOCK_PENDING
	_LOCK_EXCLUSIVE = sqlite3vfs.LOCK_EXCLUSIVE
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
