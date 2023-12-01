#include <time.h>

// Platform Configuration

#define SQLITE_OS_OTHER 1
#define SQLITE_BYTEORDER 1234

#define HAVE_INT8_T 1
#define HAVE_INT16_T 1
#define HAVE_INT32_T 1
#define HAVE_INT64_T 1
#define HAVE_UINT8_T 1
#define HAVE_UINT16_T 1
#define HAVE_UINT32_T 1
#define HAVE_UINT64_T 1
#define HAVE_STDINT_H 1
#define HAVE_INTTYPES_H 1

#define HAVE_LOG2 1
#define HAVE_LOG10 1
#define HAVE_ISNAN 1

#define HAVE_USLEEP 1
#define HAVE_NANOSLEEP 1

#define HAVE_GMTIME_R 1
#define HAVE_LOCALTIME_S 1

#define HAVE_MALLOC_H 1
#define HAVE_MALLOC_USABLE_SIZE 1

// Recommended Options

#define SQLITE_DQS 0
#define SQLITE_THREADSAFE 0
#define SQLITE_DEFAULT_MEMSTATUS 0
#define SQLITE_DEFAULT_WAL_SYNCHRONOUS 1
#define SQLITE_LIKE_DOESNT_MATCH_BLOBS
#define SQLITE_MAX_EXPR_DEPTH 0
#define SQLITE_USE_ALLOCA
#define SQLITE_OMIT_DEPRECATED
#define SQLITE_OMIT_SHARED_CACHE
#define SQLITE_OMIT_AUTOINIT
// #define SQLITE_OMIT_DECLTYPE
// #define SQLITE_OMIT_PROGRESS_CALLBACK

// Other Options

#define SQLITE_ALLOW_URI_AUTHORITY
#define SQLITE_TRUSTED_SCHEMA 0
#define SQLITE_DEFAULT_FOREIGN_KEYS 1
#define SQLITE_ENABLE_ATOMIC_WRITE
#define SQLITE_ENABLE_BATCH_ATOMIC_WRITE

// Because WASM does not support shared memory,
// SQLite disables WAL for WASM builds.
// We patch SQLite to use exclusive locking mode instead.
// https://sqlite.org/wal.html#noshm
#undef SQLITE_OMIT_WAL

// We have our own memdb VFS.
// To avoid interactions between the two,
// omit sqlite3_serialize/sqlite3_deserialize,
// which we also don't wrap.
#define SQLITE_OMIT_DESERIALIZE

// Amalgamated Extensions

#define SQLITE_ENABLE_MATH_FUNCTIONS 1
#define SQLITE_ENABLE_JSON1 1
#define SQLITE_ENABLE_FTS3 1
#define SQLITE_ENABLE_FTS3_PARENTHESIS 1
#define SQLITE_ENABLE_FTS4 1
#define SQLITE_ENABLE_FTS5 1
#define SQLITE_ENABLE_RTREE 1
#define SQLITE_ENABLE_GEOPOLY 1

#define SQLITE_SOUNDEX

// Session Extension
// #define SQLITE_ENABLE_SESSION
// #define SQLITE_ENABLE_PREUPDATE_HOOK

// Implemented in vfs.c.
int localtime_s(struct tm *const pTm, time_t const *const pTime);