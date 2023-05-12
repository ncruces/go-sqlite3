#include <stdbool.h>
#include <stddef.h>

#include "sqlite_cfg.h"
//
#include "sqlite3.c"
//
#include "os.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);

int sqlite3_os_init() {
  return sqlite3_vfs_register(os_vfs(), /*default=*/true);
}

#define randomFunc(args...) randomFunc2(args)
#include "speedtest1.c"
