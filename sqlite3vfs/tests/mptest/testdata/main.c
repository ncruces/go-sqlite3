#include <stdbool.h>
#include <stddef.h>

// Configuration
#include "sqlite_cfg.h"
// Amalgamation
#include "sqlite3.c"
// VFS
#include "vfs.c"

__attribute__((constructor)) void init() { sqlite3_initialize(); }

static int dont_unlink(const char *pathname) { return 0; }
#define sqlite3_enable_load_extension(...)
#define sqlite3_trace(...)
#define unlink dont_unlink
#undef UNUSED_PARAMETER
#include "mptest.c"