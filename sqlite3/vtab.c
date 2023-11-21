#include <stddef.h>

#include "include.h"
#include "sqlite3.h"

#define SQLITE_VTAB_CREATOR_GO /******/ 0x01
#define SQLITE_VTAB_DESTROYER_GO /****/ 0x02
#define SQLITE_VTAB_UPDATER_GO /******/ 0x04
#define SQLITE_VTAB_RENAMER_GO /******/ 0x08
#define SQLITE_VTAB_OVERLOADER_GO /***/ 0x10
#define SQLITE_VTAB_CHECKER_GO /******/ 0x20
#define SQLITE_VTAB_TX_GO /***********/ 0x40
#define SQLITE_VTAB_SAVEPOINTER_GO /**/ 0x80

int go_vtab_create(sqlite3_module *, int argc, const char *const *argv,
                   sqlite3_vtab **, char **pzErr);
int go_vtab_connect(sqlite3_module *, int argc, const char *const *argv,
                    sqlite3_vtab **, char **pzErr);

int go_vtab_disconnect(sqlite3_vtab *);
int go_vtab_destroy(sqlite3_vtab *);
int go_vtab_best_index(sqlite3_vtab *, sqlite3_index_info *);
int go_cur_open(sqlite3_vtab *, sqlite3_vtab_cursor **);

int go_cur_close(sqlite3_vtab_cursor *);
int go_cur_filter(sqlite3_vtab_cursor *, int idxNum, const char *idxStr,
                  int argc, sqlite3_value **argv);
int go_cur_next(sqlite3_vtab_cursor *);
int go_cur_eof(sqlite3_vtab_cursor *);
int go_cur_column(sqlite3_vtab_cursor *, sqlite3_context *, int);
int go_cur_rowid(sqlite3_vtab_cursor *, sqlite3_int64 *pRowid);

int go_vtab_update(sqlite3_vtab *, int, sqlite3_value **, sqlite3_int64 *);
int go_vtab_rename(sqlite3_vtab *, const char *zNew);
int go_vtab_find_function(sqlite3_vtab *, int nArg, const char *zName,
                          go_handle *pxFunc);

int go_vtab_begin(sqlite3_vtab *);
int go_vtab_sync(sqlite3_vtab *);
int go_vtab_commit(sqlite3_vtab *);
int go_vtab_rollback(sqlite3_vtab *);

int go_vtab_savepoint(sqlite3_vtab *, int);
int go_vtab_release(sqlite3_vtab *, int);
int go_vtab_rollback_to(sqlite3_vtab *, int);

int go_vtab_integrity(sqlite3_vtab *, const char *zSchema, const char *zTabName,
                      int mFlags, char **pzErr);

struct go_module {
  go_handle handle;
  sqlite3_module base;
};

struct go_vtab {
  go_handle handle;
  sqlite3_vtab base;
};

struct go_cursor {
  go_handle handle;
  sqlite3_vtab_cursor base;
};

static void go_mod_destroy(void *pAux) {
  struct go_module *mod = pAux;
  void *handle = mod->handle;
  free(mod);
  go_destroy(handle);
}

static int go_vtab_create_wrapper(sqlite3 *db, void *pAux, int argc,
                                  const char *const *argv,
                                  sqlite3_vtab **ppVTab, char **pzErr) {
  struct go_vtab *vtab = calloc(1, sizeof(struct go_vtab));
  if (vtab == NULL) return SQLITE_NOMEM;
  *ppVTab = &vtab->base;

  struct go_module *mod = pAux;
  int rc = go_vtab_create(&mod->base, argc, argv, ppVTab, pzErr);
  if (rc) {
    if (*pzErr) *pzErr = sqlite3_mprintf("%s", *pzErr);
    free(vtab);
  }
  return rc;
}

static int go_vtab_connect_wrapper(sqlite3 *db, void *pAux, int argc,
                                   const char *const *argv,
                                   sqlite3_vtab **ppVTab, char **pzErr) {
  struct go_vtab *vtab = calloc(1, sizeof(struct go_vtab));
  if (vtab == NULL) return SQLITE_NOMEM;
  *ppVTab = &vtab->base;

  struct go_module *mod = pAux;
  int rc = go_vtab_connect(&mod->base, argc, argv, ppVTab, pzErr);
  if (rc) {
    free(vtab);
    if (*pzErr) *pzErr = sqlite3_mprintf("%s", *pzErr);
  }
  return rc;
}

static int go_vtab_disconnect_wrapper(sqlite3_vtab *pVTab) {
  struct go_vtab *vtab = container_of(pVTab, struct go_vtab, base);
  int rc = go_vtab_disconnect(pVTab);
  free(vtab);
  return rc;
}

static int go_vtab_destroy_wrapper(sqlite3_vtab *pVTab) {
  struct go_vtab *vtab = container_of(pVTab, struct go_vtab, base);
  int rc = go_vtab_destroy(pVTab);
  free(vtab);
  return rc;
}

static int go_cur_open_wrapper(sqlite3_vtab *pVTab,
                               sqlite3_vtab_cursor **ppCursor) {
  struct go_cursor *cur = calloc(1, sizeof(struct go_cursor));
  if (cur == NULL) return SQLITE_NOMEM;
  *ppCursor = &cur->base;

  int rc = go_cur_open(pVTab, ppCursor);
  if (rc) free(cur);
  return rc;
}

static int go_cur_close_wrapper(sqlite3_vtab_cursor *pCursor) {
  struct go_cursor *cur = container_of(pCursor, struct go_cursor, base);
  int rc = go_cur_close(pCursor);
  free(cur);
  return rc;
}

static int go_vtab_find_function_wrapper(
    sqlite3_vtab *pVTab, int nArg, const char *zName,
    void (**pxFunc)(sqlite3_context *, int, sqlite3_value **), void **ppArg) {
  struct go_vtab *vtab = container_of(pVTab, struct go_vtab, base);

  go_handle handle;
  int rc = go_vtab_find_function(pVTab, nArg, zName, &handle);
  if (rc) {
    *pxFunc = go_func;
    *ppArg = handle;
  }
  return rc;
}

static int go_vtab_integrity_wrapper(sqlite3_vtab *pVTab, const char *zSchema,
                                     const char *zTabName, int mFlags,
                                     char **pzErr) {
  int rc = go_vtab_integrity(pVTab, zSchema, zTabName, mFlags, pzErr);
  if (rc && *pzErr) *pzErr = sqlite3_mprintf("%s", *pzErr);
  return rc;
}

int sqlite3_create_module_go(sqlite3 *db, const char *zName, int flags,
                             go_handle handle) {
  struct go_module *mod = malloc(sizeof(struct go_module));
  if (mod == NULL) {
    go_destroy(handle);
    return SQLITE_NOMEM;
  }

  mod->handle = handle;
  mod->base = (sqlite3_module){
      .iVersion = 4,
      .xConnect = go_vtab_connect_wrapper,
      .xDisconnect = go_vtab_disconnect_wrapper,
      .xBestIndex = go_vtab_best_index,
      .xOpen = go_cur_open_wrapper,
      .xClose = go_cur_close_wrapper,
      .xFilter = go_cur_filter,
      .xNext = go_cur_next,
      .xEof = go_cur_eof,
      .xColumn = go_cur_column,
      .xRowid = go_cur_rowid,
  };
  if (flags & SQLITE_VTAB_CREATOR_GO) {
    if (flags & SQLITE_VTAB_DESTROYER_GO) {
      mod->base.xCreate = go_vtab_create_wrapper;
      mod->base.xDestroy = go_vtab_destroy_wrapper;
    } else {
      mod->base.xCreate = mod->base.xConnect;
      mod->base.xDestroy = mod->base.xDisconnect;
    }
  }
  if (flags & SQLITE_VTAB_UPDATER_GO) {
    mod->base.xUpdate = go_vtab_update;
  }
  if (flags & SQLITE_VTAB_RENAMER_GO) {
    mod->base.xRename = go_vtab_rename;
  }
  if (flags & SQLITE_VTAB_OVERLOADER_GO) {
    mod->base.xFindFunction = go_vtab_find_function_wrapper;
  }
  if (flags & SQLITE_VTAB_CHECKER_GO) {
    mod->base.xIntegrity = go_vtab_integrity_wrapper;
  }
  if (flags & SQLITE_VTAB_TX_GO) {
    mod->base.xBegin = go_vtab_begin;
    mod->base.xSync = go_vtab_sync;
    mod->base.xCommit = go_vtab_commit;
    mod->base.xRollback = go_vtab_rollback;
  }
  if (flags & SQLITE_VTAB_SAVEPOINTER_GO) {
    mod->base.xSavepoint = go_vtab_savepoint;
    mod->base.xRelease = go_vtab_release;
    mod->base.xRollbackTo = go_vtab_rollback_to;
  }

  return sqlite3_create_module_v2(db, zName, &mod->base, mod, go_mod_destroy);
}

int sqlite3_vtab_config_go(sqlite3 *db, int op, int constraint) {
  return sqlite3_vtab_config(db, op, constraint);
}

static_assert(offsetof(struct go_module, base) == 4, "Unexpected offset");
static_assert(offsetof(struct go_vtab, base) == 4, "Unexpected offset");
static_assert(offsetof(struct go_cursor, base) == 4, "Unexpected offset");
static_assert(sizeof(struct sqlite3_index_info) == 72, "Unexpected size");
static_assert(sizeof(struct sqlite3_index_orderby) == 8, "Unexpected size");
static_assert(sizeof(struct sqlite3_index_constraint) == 12, "Unexpected size");
static_assert(sizeof(struct sqlite3_index_constraint_usage) == 8,
              "Unexpected size");