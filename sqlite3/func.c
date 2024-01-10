#include <stddef.h>

#include "include.h"
#include "sqlite3.h"

int go_compare(go_handle, int, const void *, int, const void *);

void go_func(sqlite3_context *, go_handle, int, sqlite3_value **);

void go_step(sqlite3_context *, go_handle *, go_handle, int, sqlite3_value **);
void go_inverse(sqlite3_context *, go_handle *, int, sqlite3_value **);
void go_value(sqlite3_context *, go_handle);
void go_final(sqlite3_context *, go_handle, go_handle);

void go_func_wrapper(sqlite3_context *ctx, int nArg, sqlite3_value **pArg) {
  go_func(ctx, sqlite3_user_data(ctx), nArg, pArg);
}

void go_step_wrapper(sqlite3_context *ctx, int nArg, sqlite3_value **pArg) {
  go_handle *agg = sqlite3_aggregate_context(ctx, 4);
  go_handle data = NULL;
  if (agg && *agg == NULL) {
    data = sqlite3_user_data(ctx);
  }
  go_step(ctx, agg, data, nArg, pArg);
}

void go_inverse_wrapper(sqlite3_context *ctx, int nArg, sqlite3_value **pArg) {
  go_handle *agg = sqlite3_aggregate_context(ctx, 4);
  go_inverse(ctx, *agg, nArg, pArg);
}

void go_value_wrapper(sqlite3_context *ctx) {
  go_handle *agg = sqlite3_aggregate_context(ctx, 4);
  go_value(ctx, *agg);
}

void go_final_wrapper(sqlite3_context *ctx) {
  go_handle *agg = sqlite3_aggregate_context(ctx, 0);
  go_handle data = NULL;
  if (agg == NULL || *agg == NULL) {
    data = sqlite3_user_data(ctx);
  }
  go_final(ctx, agg, data);
}

int sqlite3_create_collation_go(sqlite3 *db, const char *name, go_handle app) {
  int rc = sqlite3_create_collation_v2(db, name, SQLITE_UTF8, app, go_compare,
                                       go_destroy);
  if (rc) go_destroy(app);
  return rc;
}

int sqlite3_create_function_go(sqlite3 *db, const char *name, int argc,
                               int flags, go_handle app) {
  return sqlite3_create_function_v2(db, name, argc, SQLITE_UTF8 | flags, app,
                                    go_func_wrapper, /*step=*/NULL,
                                    /*final=*/NULL, go_destroy);
}

int sqlite3_create_aggregate_function_go(sqlite3 *db, const char *name,
                                         int argc, int flags, go_handle app) {
  return sqlite3_create_window_function(db, name, argc, SQLITE_UTF8 | flags,
                                        app, go_step_wrapper, go_final_wrapper,
                                        /*value=*/NULL, /*inverse=*/NULL,
                                        go_destroy);
}

int sqlite3_create_window_function_go(sqlite3 *db, const char *name, int argc,
                                      int flags, go_handle app) {
  return sqlite3_create_window_function(
      db, name, argc, SQLITE_UTF8 | flags, app, go_step_wrapper,
      go_final_wrapper, go_value_wrapper, go_inverse_wrapper, go_destroy);
}

void sqlite3_set_auxdata_go(sqlite3_context *ctx, int i, go_handle aux) {
  sqlite3_set_auxdata(ctx, i, aux, go_destroy);
}