package vfs

const (
	_MAX_PATHNAME = 512

	ptrlen = 4
)

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
	_OK_SYMLINK              = (_OK | (2 << 8)) /* internal use only */
)

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

type _AccessFlag uint32

const (
	_ACCESS_EXISTS    _AccessFlag = 0
	_ACCESS_READWRITE _AccessFlag = 1 /* Used by PRAGMA temp_store_directory */
	_ACCESS_READ      _AccessFlag = 2 /* Unused */
)

type _SyncFlag uint32

const (
	_SYNC_NORMAL   _SyncFlag = 0x00002
	_SYNC_FULL     _SyncFlag = 0x00003
	_SYNC_DATAONLY _SyncFlag = 0x00010
)

type _FcntlOpcode uint32

const (
	_FCNTL_LOCKSTATE             = 1
	_FCNTL_GET_LOCKPROXYFILE     = 2
	_FCNTL_SET_LOCKPROXYFILE     = 3
	_FCNTL_LAST_ERRNO            = 4
	_FCNTL_SIZE_HINT             = 5
	_FCNTL_CHUNK_SIZE            = 6
	_FCNTL_FILE_POINTER          = 7
	_FCNTL_SYNC_OMITTED          = 8
	_FCNTL_WIN32_AV_RETRY        = 9
	_FCNTL_PERSIST_WAL           = 10
	_FCNTL_OVERWRITE             = 11
	_FCNTL_VFSNAME               = 12
	_FCNTL_POWERSAFE_OVERWRITE   = 13
	_FCNTL_PRAGMA                = 14
	_FCNTL_BUSYHANDLER           = 15
	_FCNTL_TEMPFILENAME          = 16
	_FCNTL_MMAP_SIZE             = 18
	_FCNTL_TRACE                 = 19
	_FCNTL_HAS_MOVED             = 20
	_FCNTL_SYNC                  = 21
	_FCNTL_COMMIT_PHASETWO       = 22
	_FCNTL_WIN32_SET_HANDLE      = 23
	_FCNTL_WAL_BLOCK             = 24
	_FCNTL_ZIPVFS                = 25
	_FCNTL_RBU                   = 26
	_FCNTL_VFS_POINTER           = 27
	_FCNTL_JOURNAL_POINTER       = 28
	_FCNTL_WIN32_GET_HANDLE      = 29
	_FCNTL_PDB                   = 30
	_FCNTL_BEGIN_ATOMIC_WRITE    = 31
	_FCNTL_COMMIT_ATOMIC_WRITE   = 32
	_FCNTL_ROLLBACK_ATOMIC_WRITE = 33
	_FCNTL_LOCK_TIMEOUT          = 34
	_FCNTL_DATA_VERSION          = 35
	_FCNTL_SIZE_LIMIT            = 36
	_FCNTL_CKPT_DONE             = 37
	_FCNTL_RESERVE_BYTES         = 38
	_FCNTL_CKPT_START            = 39
	_FCNTL_EXTERNAL_READER       = 40
	_FCNTL_CKSM_FILE             = 41
	_FCNTL_RESET_CACHE           = 42
)
