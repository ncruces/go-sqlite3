#include <stdbool.h>
#include <stddef.h>

#include "sqlite3.c"
//
#include "os.c"
#include "qsort.c"
#include "time.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);

int sqlite3_os_init() {
  return sqlite3_vfs_register(os_vfs(), /*default=*/true);
}

int main() {
  int rc = sqlite3_initialize();
  if (rc != SQLITE_OK) return 1;
}
