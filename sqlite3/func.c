#include <stddef.h>

#include "include.h"
#include "sqlite3.h"

void go_func(sqlite3_context *, int, sqlite3_value **);
void go_step(sqlite3_context *, int, sqlite3_value **);
void go_final(sqlite3_context *);
void go_value(sqlite3_context *);
void go_inverse(sqlite3_context *, int, sqlite3_value **);

int go_compare(go_handle, int, const void *, int, const void *);

int sqlite3_create_collation_go(sqlite3 *db, const char *name, go_handle app) {
  int rc = sqlite3_create_collation_v2(db, name, SQLITE_UTF8, app, go_compare,
                                       go_destroy);
  if (rc) go_destroy(app);
  return rc;
}

int sqlite3_create_function_go(sqlite3 *db, const char *name, int argc,
                               int flags, go_handle app) {
  return sqlite3_create_function_v2(db, name, argc, SQLITE_UTF8 | flags, app,
                                    go_func, /*step=*/NULL, /*final=*/NULL,
                                    go_destroy);
}

int sqlite3_create_aggregate_function_go(sqlite3 *db, const char *name,
                                         int argc, int flags, go_handle app) {
  return sqlite3_create_window_function(db, name, argc, SQLITE_UTF8 | flags,
                                        app, go_step, go_final, /*value=*/NULL,
                                        /*inverse=*/NULL, go_destroy);
}

int sqlite3_create_window_function_go(sqlite3 *db, const char *name, int argc,
                                      int flags, go_handle app) {
  return sqlite3_create_window_function(db, name, argc, SQLITE_UTF8 | flags,
                                        app, go_step, go_final, go_value,
                                        go_inverse, go_destroy);
}

void sqlite3_set_auxdata_go(sqlite3_context *ctx, int i, go_handle aux) {
  sqlite3_set_auxdata(ctx, i, aux, go_destroy);
}