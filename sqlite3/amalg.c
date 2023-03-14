#include <stddef.h>

#include "main.c"
#include "os.c"
#include "qsort.c"
#include "time.c"

#include "sqlite3.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);

int sqlite3_unlock_os_notify(sqlite3 *pBlocked, int notifyArg) {
  return sqlite3_unlock_notify(pBlocked, os_notify, (void *)(notifyArg));
}
