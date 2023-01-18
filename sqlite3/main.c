#include <stdlib.h>

#include "sqlite3.h"

int main() {
  int rc = sqlite3_initialize();
  if (rc != SQLITE_OK) return 1;
}

int go_randomness(sqlite3_vfs *, int nByte, char *zOut);
int go_sleep(sqlite3_vfs *, int microseconds);
int go_current_time(sqlite3_vfs *, double *);
int go_current_time_64(sqlite3_vfs *, sqlite3_int64 *);

int sqlite3_os_init() {
  static sqlite3_vfs go_vfs = {
      .iVersion = 2,
      .zName = "go",
      .xRandomness = go_randomness,
      .xSleep = go_sleep,
      .xCurrentTime = go_current_time,
      .xCurrentTimeInt64 = go_current_time_64,
  };
  return sqlite3_vfs_register(&go_vfs, /*default=*/1);
}

sqlite3_destructor_type malloc_destructor = &free;