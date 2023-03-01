#include <stdbool.h>

#include "sqlite3.h"

int main() {
  int rc = sqlite3_initialize();
  if (rc != SQLITE_OK) return 1;
}

sqlite3_vfs *os_vfs();

int sqlite3_os_init() {
  return sqlite3_vfs_register(os_vfs(), /*default=*/true);
}
