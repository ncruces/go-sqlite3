#include <stdbool.h>

#include "sqlite3.h"

void go_log(void *, int, const char *);

int sqlite3_config_log_go(bool enable) {
  return sqlite3_config(SQLITE_CONFIG_LOG, enable ? go_log : NULL, NULL);
}