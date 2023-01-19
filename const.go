package sqlite3

const (
	_OK   = 0   /* Successful result */
	_ROW  = 100 /* sqlite3_step() has another row ready */
	_DONE = 101 /* sqlite3_step() has finished executing */

	_UTF8 = 1

	_MAX_PATHNAME = 512
)

type ErrorCode int

const (
	ERROR      ErrorCode = 1  /* Generic error */
	INTERNAL   ErrorCode = 2  /* Internal logic error in SQLite */
	PERM       ErrorCode = 3  /* Access permission denied */
	ABORT      ErrorCode = 4  /* Callback routine requested an abort */
	BUSY       ErrorCode = 5  /* The database file is locked */
	LOCKED     ErrorCode = 6  /* A table in the database is locked */
	NOMEM      ErrorCode = 7  /* A malloc() failed */
	READONLY   ErrorCode = 8  /* Attempt to write a readonly database */
	INTERRUPT  ErrorCode = 9  /* Operation terminated by sqlite3_interrupt()*/
	IOERR      ErrorCode = 10 /* Some kind of disk I/O error occurred */
	CORRUPT    ErrorCode = 11 /* The database disk image is malformed */
	NOTFOUND   ErrorCode = 12 /* Unknown opcode in sqlite3_file_control() */
	FULL       ErrorCode = 13 /* Insertion failed because database is full */
	CANTOPEN   ErrorCode = 14 /* Unable to open the database file */
	PROTOCOL   ErrorCode = 15 /* Database lock protocol error */
	EMPTY      ErrorCode = 16 /* Internal use only */
	SCHEMA     ErrorCode = 17 /* The database schema changed */
	TOOBIG     ErrorCode = 18 /* String or BLOB exceeds size limit */
	CONSTRAINT ErrorCode = 19 /* Abort due to constraint violation */
	MISMATCH   ErrorCode = 20 /* Data type mismatch */
	MISUSE     ErrorCode = 21 /* Library used incorrectly */
	NOLFS      ErrorCode = 22 /* Uses OS features not supported on host */
	AUTH       ErrorCode = 23 /* Authorization denied */
	FORMAT     ErrorCode = 24 /* Not used */
	RANGE      ErrorCode = 25 /* 2nd parameter to sqlite3_bind out of range */
	NOTADB     ErrorCode = 26 /* File opened that is not a database file */
	NOTICE     ErrorCode = 27 /* Notifications from sqlite3_log() */
	WARNING    ErrorCode = 28 /* Warnings from sqlite3_log() */
)

type ExtendedErrorCode int

const (
	ERROR_MISSING_COLLSEQ   = ExtendedErrorCode(ERROR | (1 << 8))
	ERROR_RETRY             = ExtendedErrorCode(ERROR | (2 << 8))
	ERROR_SNAPSHOT          = ExtendedErrorCode(ERROR | (3 << 8))
	IOERR_READ              = ExtendedErrorCode(IOERR | (1 << 8))
	IOERR_SHORT_READ        = ExtendedErrorCode(IOERR | (2 << 8))
	IOERR_WRITE             = ExtendedErrorCode(IOERR | (3 << 8))
	IOERR_FSYNC             = ExtendedErrorCode(IOERR | (4 << 8))
	IOERR_DIR_FSYNC         = ExtendedErrorCode(IOERR | (5 << 8))
	IOERR_TRUNCATE          = ExtendedErrorCode(IOERR | (6 << 8))
	IOERR_FSTAT             = ExtendedErrorCode(IOERR | (7 << 8))
	IOERR_UNLOCK            = ExtendedErrorCode(IOERR | (8 << 8))
	IOERR_RDLOCK            = ExtendedErrorCode(IOERR | (9 << 8))
	IOERR_DELETE            = ExtendedErrorCode(IOERR | (10 << 8))
	IOERR_BLOCKED           = ExtendedErrorCode(IOERR | (11 << 8))
	IOERR_NOMEM             = ExtendedErrorCode(IOERR | (12 << 8))
	IOERR_ACCESS            = ExtendedErrorCode(IOERR | (13 << 8))
	IOERR_CHECKRESERVEDLOCK = ExtendedErrorCode(IOERR | (14 << 8))
	IOERR_LOCK              = ExtendedErrorCode(IOERR | (15 << 8))
	IOERR_CLOSE             = ExtendedErrorCode(IOERR | (16 << 8))
	IOERR_DIR_CLOSE         = ExtendedErrorCode(IOERR | (17 << 8))
	IOERR_SHMOPEN           = ExtendedErrorCode(IOERR | (18 << 8))
	IOERR_SHMSIZE           = ExtendedErrorCode(IOERR | (19 << 8))
	IOERR_SHMLOCK           = ExtendedErrorCode(IOERR | (20 << 8))
	IOERR_SHMMAP            = ExtendedErrorCode(IOERR | (21 << 8))
	IOERR_SEEK              = ExtendedErrorCode(IOERR | (22 << 8))
	IOERR_DELETE_NOENT      = ExtendedErrorCode(IOERR | (23 << 8))
	IOERR_MMAP              = ExtendedErrorCode(IOERR | (24 << 8))
	IOERR_GETTEMPPATH       = ExtendedErrorCode(IOERR | (25 << 8))
	IOERR_CONVPATH          = ExtendedErrorCode(IOERR | (26 << 8))
	IOERR_VNODE             = ExtendedErrorCode(IOERR | (27 << 8))
	IOERR_AUTH              = ExtendedErrorCode(IOERR | (28 << 8))
	IOERR_BEGIN_ATOMIC      = ExtendedErrorCode(IOERR | (29 << 8))
	IOERR_COMMIT_ATOMIC     = ExtendedErrorCode(IOERR | (30 << 8))
	IOERR_ROLLBACK_ATOMIC   = ExtendedErrorCode(IOERR | (31 << 8))
	IOERR_DATA              = ExtendedErrorCode(IOERR | (32 << 8))
	IOERR_CORRUPTFS         = ExtendedErrorCode(IOERR | (33 << 8))
	LOCKED_SHAREDCACHE      = ExtendedErrorCode(LOCKED | (1 << 8))
	LOCKED_VTAB             = ExtendedErrorCode(LOCKED | (2 << 8))
	BUSY_RECOVERY           = ExtendedErrorCode(BUSY | (1 << 8))
	BUSY_SNAPSHOT           = ExtendedErrorCode(BUSY | (2 << 8))
	BUSY_TIMEOUT            = ExtendedErrorCode(BUSY | (3 << 8))
	CANTOPEN_NOTEMPDIR      = ExtendedErrorCode(CANTOPEN | (1 << 8))
	CANTOPEN_ISDIR          = ExtendedErrorCode(CANTOPEN | (2 << 8))
	CANTOPEN_FULLPATH       = ExtendedErrorCode(CANTOPEN | (3 << 8))
	CANTOPEN_CONVPATH       = ExtendedErrorCode(CANTOPEN | (4 << 8))
	CANTOPEN_DIRTYWAL       = ExtendedErrorCode(CANTOPEN | (5 << 8)) /* Not Used */
	CANTOPEN_SYMLINK        = ExtendedErrorCode(CANTOPEN | (6 << 8))
	CORRUPT_VTAB            = ExtendedErrorCode(CORRUPT | (1 << 8))
	CORRUPT_SEQUENCE        = ExtendedErrorCode(CORRUPT | (2 << 8))
	CORRUPT_INDEX           = ExtendedErrorCode(CORRUPT | (3 << 8))
	READONLY_RECOVERY       = ExtendedErrorCode(READONLY | (1 << 8))
	READONLY_CANTLOCK       = ExtendedErrorCode(READONLY | (2 << 8))
	READONLY_ROLLBACK       = ExtendedErrorCode(READONLY | (3 << 8))
	READONLY_DBMOVED        = ExtendedErrorCode(READONLY | (4 << 8))
	READONLY_CANTINIT       = ExtendedErrorCode(READONLY | (5 << 8))
	READONLY_DIRECTORY      = ExtendedErrorCode(READONLY | (6 << 8))
	ABORT_ROLLBACK          = ExtendedErrorCode(ABORT | (2 << 8))
	CONSTRAINT_CHECK        = ExtendedErrorCode(CONSTRAINT | (1 << 8))
	CONSTRAINT_COMMITHOOK   = ExtendedErrorCode(CONSTRAINT | (2 << 8))
	CONSTRAINT_FOREIGNKEY   = ExtendedErrorCode(CONSTRAINT | (3 << 8))
	CONSTRAINT_FUNCTION     = ExtendedErrorCode(CONSTRAINT | (4 << 8))
	CONSTRAINT_NOTNULL      = ExtendedErrorCode(CONSTRAINT | (5 << 8))
	CONSTRAINT_PRIMARYKEY   = ExtendedErrorCode(CONSTRAINT | (6 << 8))
	CONSTRAINT_TRIGGER      = ExtendedErrorCode(CONSTRAINT | (7 << 8))
	CONSTRAINT_UNIQUE       = ExtendedErrorCode(CONSTRAINT | (8 << 8))
	CONSTRAINT_VTAB         = ExtendedErrorCode(CONSTRAINT | (9 << 8))
	CONSTRAINT_ROWID        = ExtendedErrorCode(CONSTRAINT | (10 << 8))
	CONSTRAINT_PINNED       = ExtendedErrorCode(CONSTRAINT | (11 << 8))
	CONSTRAINT_DATATYPE     = ExtendedErrorCode(CONSTRAINT | (12 << 8))
	NOTICE_RECOVER_WAL      = ExtendedErrorCode(NOTICE | (1 << 8))
	NOTICE_RECOVER_ROLLBACK = ExtendedErrorCode(NOTICE | (2 << 8))
	WARNING_AUTOINDEX       = ExtendedErrorCode(WARNING | (1 << 8))
	AUTH_USER               = ExtendedErrorCode(AUTH | (1 << 8))
)

type OpenFlag uint

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
	OPEN_EXRESCODE     OpenFlag = 0x02000000 /* Extended result codes */
)

type AccessFlag uint

const (
	ACCESS_EXISTS    AccessFlag = 0
	ACCESS_READWRITE AccessFlag = 1 /* Used by PRAGMA temp_store_directory */
	ACCESS_READ      AccessFlag = 2 /* Unused */
)

type PrepareFlag uint

const (
	PREPARE_PERSISTENT PrepareFlag = 0x01
	PREPARE_NORMALIZE  PrepareFlag = 0x02
	PREPARE_NO_VTAB    PrepareFlag = 0x04
)

type Datatype uint

const (
	INTEGER Datatype = 1
	FLOAT   Datatype = 2
	TEXT    Datatype = 3
	BLOB    Datatype = 4
	NULL    Datatype = 5
)
