#include <time.h>

// Platform Configuration

#define SQLITE_OS_OTHER 1
#define SQLITE_BYTEORDER 1234
#define SQLITE_MAX_PATHLEN 4096

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

#define HAVE_LOG2 1
#define HAVE_LOG10 1
#define HAVE_ISNAN 1

#define HAVE_STRCHRNUL 1
#define HAVE_NANOSLEEP 1
#define HAVE_LOCALTIME_S 1

#define HAVE_MALLOC_H 1
#define HAVE_MALLOC_USABLE_SIZE 1

// Implemented in hooks.c.
static int sqliteBusyCallback(void*, int);

// Implemented in vfs.c.
int localtime_s(struct tm* const pTm, time_t const* const pTime);
