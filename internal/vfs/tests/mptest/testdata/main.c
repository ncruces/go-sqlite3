#include <stdbool.h>
#include <stddef.h>

#include "sqlite3.c"
//
#include "os.c"
#include "qsort.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);

int sqlite3_os_init() {
  return sqlite3_vfs_register(os_vfs(), /*default=*/true);
}

__attribute__((constructor)) void premain() { sqlite3_initialize(); }

static int dont_unlink(const char *pathname) { return 0; }
#define sqlite3_enable_load_extension(...)
#define sqlite3_trace(...)
#define unlink dont_unlink
#undef UNUSED_PARAMETER
#include "mptest.c"
