#include <stdbool.h>

#include "sqlite3.h"

int go_progress_handler(void*);
int go_busy_handler(void*, int);
int go_busy_timeout(int count, int tmout);

int go_commit_hook(void*);
void go_rollback_hook(void*);
void go_update_hook(void*, int, char const*, char const*, sqlite3_int64);
int go_wal_hook(void*, sqlite3*, const char*, int);
int go_trace(unsigned, void*, void*, void*);
int go_authorizer(void*, int, const char*, const char*, const char*,
                  const char*);

void go_log(void*, int, const char*);

unsigned int go_autovacuum_pages(void*, const char*, unsigned int, unsigned int,
                                 unsigned int);

void sqlite3_log_go(int iErrCode, const char* zMsg) {
  sqlite3_log(iErrCode, "%s", zMsg);
}

void sqlite3_progress_handler_go(sqlite3* db, int n) {
  sqlite3_progress_handler(db, n, go_progress_handler, /*arg=*/NULL);
}

int sqlite3_busy_handler_go(sqlite3* db, bool enable) {
  return sqlite3_busy_handler(db, enable ? go_busy_handler : NULL, /*arg=*/db);
}

void sqlite3_commit_hook_go(sqlite3* db, bool enable) {
  sqlite3_commit_hook(db, enable ? go_commit_hook : NULL, /*arg=*/db);
}

void sqlite3_rollback_hook_go(sqlite3* db, bool enable) {
  sqlite3_rollback_hook(db, enable ? go_rollback_hook : NULL, /*arg=*/db);
}

void sqlite3_update_hook_go(sqlite3* db, bool enable) {
  sqlite3_update_hook(db, enable ? go_update_hook : NULL, /*arg=*/db);
}

void sqlite3_wal_hook_go(sqlite3* db, bool enable) {
  sqlite3_wal_hook(db, enable ? go_wal_hook : NULL, /*arg=*/NULL);
}

int sqlite3_set_authorizer_go(sqlite3* db, bool enable) {
  return sqlite3_set_authorizer(db, enable ? go_authorizer : NULL, /*arg=*/db);
}

int sqlite3_trace_go(sqlite3* db, unsigned mask) {
  return sqlite3_trace_v2(db, mask, go_trace, /*arg=*/db);
}

int sqlite3_config_log_go(bool enable) {
  return sqlite3_config(SQLITE_CONFIG_LOG, enable ? go_log : NULL,
                        /*arg=*/NULL);
}

int sqlite3_autovacuum_pages_go(sqlite3* db, go_handle app) {
  if (app == NULL) {
    return sqlite3_autovacuum_pages(db, NULL, NULL, NULL);
  }
  return sqlite3_autovacuum_pages(db, go_autovacuum_pages, app, go_destroy);
}

#ifndef sqliteBusyCallback

static int sqliteBusyCallback(void* ptr, int count) {
  return go_busy_timeout(count, ((sqlite3*)ptr)->busyTimeout);
}

#endif
