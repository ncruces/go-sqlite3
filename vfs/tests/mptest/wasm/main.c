#include <unistd.h>

// Use the default callback, not the Go one we patched in.
#define sqliteBusyCallback sqliteDefaultBusyCallback

#include "strings.c"
// Amalgamation
#include "sqlite3.c"
// VFS
#include "vfs.c"

__attribute__((constructor)) void init() { sqlite3_initialize(); }

// Ignore these.
#define sqlite3_enable_load_extension(...)
#define sqlite3_trace(...)
#define unlink(...) (0)
#undef UNUSED_PARAMETER

#include "mptest.c"