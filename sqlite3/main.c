#include <stdlib.h>

#include "sqlite3.h"

int main() {
  int rc = sqlite3_initialize();
  if (rc != SQLITE_OK) return -1;
}

sqlite3_vfs *sqlite3_demovfs();

int sqlite3_os_init() {
  return sqlite3_vfs_register(sqlite3_demovfs(), /*default=*/1);
}

sqlite3_destructor_type malloc_destructor = &free;