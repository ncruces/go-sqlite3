#include <time.h>

// Platform Configuration

#define SQLITE_OS_OTHER 1
#define SQLITE_BYTEORDER 1234

#define HAVE_INT8_T 1
#define HAVE_INT16_T 1
#define HAVE_INT32_T 1
#define HAVE_INT64_T 1
#define HAVE_INTPTR_T 1
#define HAVE_UINT8_T 1
#define HAVE_UINT16_T 1
#define HAVE_UINT32_T 1
#define HAVE_UINT64_T 1
#define HAVE_UINTPTR_T 1
#define HAVE_STDINT_H 1
#define HAVE_INTTYPES_H 1

#define LONGDOUBLE_TYPE double

#define HAVE_LOG2 1
#define HAVE_LOG10 1
#define HAVE_ISNAN 1

#define HAVE_STRCHRNUL 1

#define HAVE_USLEEP 1
#define HAVE_NANOSLEEP 1

#define HAVE_GMTIME_R 1
#define HAVE_LOCALTIME_S 1

#define HAVE_MALLOC_H 1
#define HAVE_MALLOC_USABLE_SIZE 1

// Because Wasm does not support shared memory,
// SQLite disables WAL for Wasm builds.
#undef SQLITE_OMIT_WAL

// Implemented in vfs.c.
int localtime_s(struct tm *const pTm, time_t const *const pTime);

// Implemented in hooks.c.
#ifndef sqliteBusyCallback
static int sqliteBusyCallback(sqlite3 *, int);
#endif