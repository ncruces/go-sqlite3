#include <stdbool.h>

#include "sqlite3.h"

int go_progress(void *);

int go_commit_hook(void *);
void go_rollback_hook(void *);
void go_update_hook(void *, int, char const *, char const *, sqlite3_int64);

int go_authorizer(void *, int, const char *, const char *, const char *,
                  const char *);

void go_log(void *, int, const char *);

void sqlite3_progress_handler_go(sqlite3 *db, int n) {
  sqlite3_progress_handler(db, n, go_progress, /*arg=*/db);
}

void sqlite3_commit_hook_go(sqlite3 *db, bool enable) {
  sqlite3_commit_hook(db, enable ? go_commit_hook : NULL, /*arg=*/db);
}

void sqlite3_rollback_hook_go(sqlite3 *db, bool enable) {
  sqlite3_rollback_hook(db, enable ? go_rollback_hook : NULL, /*arg=*/db);
}

void sqlite3_update_hook_go(sqlite3 *db, bool enable) {
  sqlite3_update_hook(db, enable ? go_update_hook : NULL, /*arg=*/db);
}

int sqlite3_set_authorizer_go(sqlite3 *db, bool enable) {
  return sqlite3_set_authorizer(db, enable ? go_authorizer : NULL, /*arg=*/db);
}

int sqlite3_config_log_go(bool enable) {
  return sqlite3_config(SQLITE_CONFIG_LOG, enable ? go_log : NULL,
                        /*arg=*/NULL);
}