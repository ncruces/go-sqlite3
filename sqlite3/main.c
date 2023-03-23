#include <stdbool.h>
#include <stddef.h>

#include "sqlite3.c"
//
#include "os.c"
#include "qsort.c"
//
#include "ext/base64.c"
#include "ext/decimal.c"
#include "ext/regexp.c"
#include "ext/series.c"
#include "ext/uint.c"
#include "ext/uuid.c"
#include "time.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);

int sqlite3_os_init() {
  return sqlite3_vfs_register(os_vfs(), /*default=*/true);
}

int main() {
  int rc = sqlite3_initialize();
  if (rc != SQLITE_OK) return 1;

  sqlite3_auto_extension((void (*)(void))sqlite3_base_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_decimal_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_regexp_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_series_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_uint_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_uuid_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_time_init);
}
