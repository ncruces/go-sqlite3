#include <time.h>

// Platform Configuration

#define SQLITE_OS_OTHER 1
#define SQLITE_BYTEORDER 1234

#define HAVE_STDINT_H 1
#define HAVE_INTTYPES_H 1

#define HAVE_ISNAN 1
#define HAVE_USLEEP 1
#define HAVE_LOCALTIME_S 1
#define HAVE_MALLOC_USABLE_SIZE 1

// Recommended Options

#define SQLITE_DQS 0
#define SQLITE_THREADSAFE 0
#define SQLITE_DEFAULT_MEMSTATUS 0
#define SQLITE_DEFAULT_WAL_SYNCHRONOUS 1
#define SQLITE_LIKE_DOESNT_MATCH_BLOBS
#define SQLITE_MAX_EXPR_DEPTH 0
#define SQLITE_OMIT_DECLTYPE
#define SQLITE_OMIT_DEPRECATED
#define SQLITE_OMIT_PROGRESS_CALLBACK
#define SQLITE_OMIT_SHARED_CACHE
#define SQLITE_OMIT_AUTOINIT
#define SQLITE_USE_ALLOCA

// Other Options
#define SQLITE_ALLOW_URI_AUTHORITY

// Because WASM does not support shared memory,
// SQLite disables WAL for WASM builds.
// We set the default locking mode to EXCLUSIVE instead.
// https://www.sqlite.org/wal.html#noshm
#undef SQLITE_OMIT_WAL
#ifndef SQLITE_DEFAULT_LOCKING_MODE
#define SQLITE_DEFAULT_LOCKING_MODE 1
#endif

// Amalgamated Extensions

#define SQLITE_ENABLE_MATH_FUNCTIONS 1
#define SQLITE_ENABLE_JSON1 1
#define SQLITE_ENABLE_FTS3 1
#define SQLITE_ENABLE_FTS3_PARENTHESIS 1
#define SQLITE_ENABLE_FTS4 1
#define SQLITE_ENABLE_FTS5 1
#define SQLITE_ENABLE_RTREE 1
#define SQLITE_ENABLE_GEOPOLY 1

// Session Extension
// #define SQLITE_ENABLE_SESSION
// #define SQLITE_ENABLE_PREUPDATE_HOOK

// https://stackoverflow.com/a/50616684
#define SECOND(...) SECOND_I(__VA_ARGS__, , )
#define SECOND_I(A, B, ...) B
#define GLUE(A, B) GLUE_I(A, B)
#define GLUE_I(A, B) A##B
#define REPLACE_DEFAULT(a, prefix) SECOND(GLUE(prefix, __LINE__), a)
#define REPLACE_AT_LINE(a) , a

// Implemented in vfs.c.
int localtime_s(struct tm *const pTm, time_t const *const pTime);