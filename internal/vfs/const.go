package vfs

import (
	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

const (
	_MAX_STRING          = 512 // Used for short strings: names, error messages…
	_MAX_PATHNAME        = 512
	_DEFAULT_SECTOR_SIZE = 4096
)

// https://www.sqlite.org/rescode.html
type _ErrorCode uint32

func (e _ErrorCode) Error() string {
	return util.ErrorCodeString(uint32(e))
}

const (
	_OK                      _ErrorCode = util.OK
	_PERM                    _ErrorCode = util.PERM
	_BUSY                    _ErrorCode = util.BUSY
	_IOERR                   _ErrorCode = util.IOERR
	_NOTFOUND                _ErrorCode = util.NOTFOUND
	_CANTOPEN                _ErrorCode = util.CANTOPEN
	_IOERR_READ              _ErrorCode = util.IOERR_READ
	_IOERR_SHORT_READ        _ErrorCode = util.IOERR_SHORT_READ
	_IOERR_WRITE             _ErrorCode = util.IOERR_WRITE
	_IOERR_FSYNC             _ErrorCode = util.IOERR_FSYNC
	_IOERR_DIR_FSYNC         _ErrorCode = util.IOERR_DIR_FSYNC
	_IOERR_TRUNCATE          _ErrorCode = util.IOERR_TRUNCATE
	_IOERR_FSTAT             _ErrorCode = util.IOERR_FSTAT
	_IOERR_UNLOCK            _ErrorCode = util.IOERR_UNLOCK
	_IOERR_RDLOCK            _ErrorCode = util.IOERR_RDLOCK
	_IOERR_DELETE            _ErrorCode = util.IOERR_DELETE
	_IOERR_ACCESS            _ErrorCode = util.IOERR_ACCESS
	_IOERR_CHECKRESERVEDLOCK _ErrorCode = util.IOERR_CHECKRESERVEDLOCK
	_IOERR_LOCK              _ErrorCode = util.IOERR_LOCK
	_IOERR_CLOSE             _ErrorCode = util.IOERR_CLOSE
	_IOERR_SEEK              _ErrorCode = util.IOERR_SEEK
	_IOERR_DELETE_NOENT      _ErrorCode = util.IOERR_DELETE_NOENT
	_CANTOPEN_FULLPATH       _ErrorCode = util.CANTOPEN_FULLPATH
	_OK_SYMLINK              _ErrorCode = util.OK_SYMLINK
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

// https://www.sqlite.org/c3ref/c_iocap_atomic.html
type _DeviceCharacteristic = sqlite3vfs.DeviceCharacteristic

const (
	_IOCAP_ATOMIC                = sqlite3vfs.IOCAP_ATOMIC
	_IOCAP_ATOMIC512             = sqlite3vfs.IOCAP_ATOMIC512
	_IOCAP_ATOMIC1K              = sqlite3vfs.IOCAP_ATOMIC1K
	_IOCAP_ATOMIC2K              = sqlite3vfs.IOCAP_ATOMIC2K
	_IOCAP_ATOMIC4K              = sqlite3vfs.IOCAP_ATOMIC4K
	_IOCAP_ATOMIC8K              = sqlite3vfs.IOCAP_ATOMIC8K
	_IOCAP_ATOMIC16K             = sqlite3vfs.IOCAP_ATOMIC16K
	_IOCAP_ATOMIC32K             = sqlite3vfs.IOCAP_ATOMIC32K
	_IOCAP_ATOMIC64K             = sqlite3vfs.IOCAP_ATOMIC64K
	_IOCAP_SAFE_APPEND           = sqlite3vfs.IOCAP_SAFE_APPEND
	_IOCAP_SEQUENTIAL            = sqlite3vfs.IOCAP_SEQUENTIAL
	_IOCAP_UNDELETABLE_WHEN_OPEN = sqlite3vfs.IOCAP_UNDELETABLE_WHEN_OPEN
	_IOCAP_POWERSAFE_OVERWRITE   = sqlite3vfs.IOCAP_POWERSAFE_OVERWRITE
	_IOCAP_IMMUTABLE             = sqlite3vfs.IOCAP_IMMUTABLE
	_IOCAP_BATCH_ATOMIC          = sqlite3vfs.IOCAP_BATCH_ATOMIC
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
