#include <stdlib.h>

#include "sqlite3.h"

int sqlite3_bind_text_go(sqlite3_stmt* stmt, int i, const char* zData,
                         sqlite3_uint64 nData) {
  return sqlite3_bind_text64(stmt, i, zData, nData, &sqlite3_free, SQLITE_UTF8);
}

int sqlite3_bind_blob_go(sqlite3_stmt* stmt, int i, const char* zData,
                         sqlite3_uint64 nData) {
  return sqlite3_bind_blob64(stmt, i, zData, nData, &sqlite3_free);
}