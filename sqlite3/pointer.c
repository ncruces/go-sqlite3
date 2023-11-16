
#include "sqlite3.h"
#include "types.h"

#define GO_POINTER_TYPE "github.com/ncruces/go-sqlite3.Pointer"

int sqlite3_bind_pointer_go(sqlite3_stmt *stmt, int i, go_handle app) {
  return sqlite3_bind_pointer(stmt, i, app, GO_POINTER_TYPE, go_destroy);
}

void sqlite3_result_pointer_go(sqlite3_context *ctx, go_handle app) {
  sqlite3_result_pointer(ctx, app, GO_POINTER_TYPE, go_destroy);
}

go_handle sqlite3_value_pointer_go(sqlite3_value *val) {
  return sqlite3_value_pointer(val, GO_POINTER_TYPE);
}