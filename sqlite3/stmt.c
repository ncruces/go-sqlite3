#include <stddef.h>

#include "sqlite3.h"

int sqlite3_exec_go(sqlite3_stmt *stmt) {
  while (sqlite3_step(stmt) == SQLITE_ROW);
  return sqlite3_reset(stmt);
}

union sqlite3_data {
  sqlite3_int64 i;
  double d;
  struct {
    const void *ptr;
    int len;
  };
};

int sqlite3_columns_go(sqlite3_stmt *stmt, int nCol, char *aType,
                       union sqlite3_data *aData) {
  if (nCol != sqlite3_column_count(stmt)) {
    return SQLITE_MISUSE;
  }
  bool check = false;
  for (int i = 0; i < nCol; ++i) {
    const void *ptr = NULL;
    switch (aType[i] = sqlite3_column_type(stmt, i)) {
      default:  // SQLITE_NULL
        aData[i] = (union sqlite3_data){};
        continue;
      case SQLITE_INTEGER:
        aData[i].i = sqlite3_column_int64(stmt, i);
        continue;
      case SQLITE_FLOAT:
        aData[i].d = sqlite3_column_double(stmt, i);
        continue;
      case SQLITE_TEXT:
        ptr = sqlite3_column_text(stmt, i);
        break;
      case SQLITE_BLOB:
        ptr = sqlite3_column_blob(stmt, i);
        break;
    }
    aData[i].ptr = ptr;
    aData[i].len = sqlite3_column_bytes(stmt, i);
    if (ptr == NULL) check = true;
  }
  if (check && SQLITE_NOMEM == sqlite3_errcode(sqlite3_db_handle(stmt))) {
    return SQLITE_NOMEM;
  }
  return SQLITE_OK;
}

static_assert(offsetof(union sqlite3_data, i) == 0, "Unexpected offset");
static_assert(offsetof(union sqlite3_data, d) == 0, "Unexpected offset");
static_assert(offsetof(union sqlite3_data, ptr) == 0, "Unexpected offset");
static_assert(offsetof(union sqlite3_data, len) == 4, "Unexpected offset");
static_assert(sizeof(union sqlite3_data) == 8, "Unexpected size");