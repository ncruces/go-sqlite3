#include <time.h>

// Platform Configuration

#define SQLITE_OS_OTHER 1
#define SQLITE_BYTEORDER 1234

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

// Need this to access WAL databases without the use of shared memory.
#define SQLITE_DEFAULT_LOCKING_MODE 1

// Implemented in Go.
int localtime_s(struct tm *const pTm, time_t const *const pTime);