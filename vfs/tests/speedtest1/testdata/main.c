#define sqliteBusyCallback sqliteDefaultBusyCallback

// Amalgamation
#include "sqlite3.c"
// VFS
#include "vfs.c"

#define randomFunc randomFunc2
#include "speedtest1.c"