#include <unistd.h>

#define sqliteBusyCallback sqliteDefaultBusyCallback

// Amalgamation
#include "sqlite3.c"
// VFS
#include "vfs.c"

__attribute__((constructor)) void init() { sqlite3_initialize(); }

#define sqlite3_enable_load_extension(...)
#define sqlite3_trace(...)
#define unlink(...) (0)
#undef UNUSED_PARAMETER
#include "mptest.c"