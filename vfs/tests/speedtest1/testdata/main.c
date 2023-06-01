#include <stdbool.h>
#include <stddef.h>

// Configuration
#include "sqlite_cfg.h"
// Amalgamation
#include "sqlite3.c"
// VFS
#include "vfs.c"

#define randomFunc(args...) randomFunc2(args)
#include "speedtest1.c"