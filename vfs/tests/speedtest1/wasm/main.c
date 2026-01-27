// Use the default callback, not the Go one we patched in.
#define sqliteBusyCallback sqliteDefaultBusyCallback

// Amalgamation
#include "sqlite3.c"
// VFS
#include "vfs.c"

// Can't have two functions with the same name.
#define randomFunc randomFuncRepeatable

#include "speedtest1.c"
