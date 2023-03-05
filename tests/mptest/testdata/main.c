#include <stdbool.h>
#include <stddef.h>

#include "os.c"
#include "qsort.c"
#include "sqlite3.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);

void __attribute__((constructor)) premain() { sqlite3_initialize(); }

int sqlite3_enable_load_extension(sqlite3 *db, int onoff) { return SQLITE_OK; }

void *sqlite3_trace(sqlite3 *db, void (*xTrace)(void *, const char *),
                    void *pArg) {
  return NULL;
}

int sqlite3_os_init() {
  return sqlite3_vfs_register(os_vfs(), /*default=*/true);
}