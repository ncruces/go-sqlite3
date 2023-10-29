#include <stddef.h>
#include <string.h>

#include "sqlite3.h"

static int time_collation(void *pArg, int nKey1, const void *pKey1, int nKey2,
                          const void *pKey2) {
  // Remove a Z suffix if one key is no longer than the other.
  // A Z suffix collates before any character but after the empty string.
  // This avoids making different keys equal.
  const int nK1 = nKey1;
  const int nK2 = nKey2;
  const char *pK1 = (const char *)pKey1;
  const char *pK2 = (const char *)pKey2;
  if (nK1 && nK1 <= nK2 && pK1[nK1 - 1] == 'Z') {
    nKey1--;
  }
  if (nK2 && nK2 <= nK1 && pK2[nK2 - 1] == 'Z') {
    nKey2--;
  }

  int n = nKey1 < nKey2 ? nKey1 : nKey2;
  int rc = memcmp(pKey1, pKey2, n);
  if (rc == 0) {
    rc = nKey1 - nKey2;
  }
  return rc;
}

static void json_time_func(sqlite3_context *context, int argc,
                          sqlite3_value **argv) {
  DateTime x;
  if (isDate(context, argc, argv, &x)) return;
  if (x.tzSet && x.tz) {
    x.iJD += x.tz * 60000;
    if (!validJulianDay(x.iJD)) return;
    x.validYMD = 0;
    x.validHMS = 0;
  }
  computeYMD_HMS(&x);

  sqlite3 *db = sqlite3_context_db_handle(context);
  sqlite3_str *res = sqlite3_str_new(db);

  sqlite3_str_appendf(res, "%04d-%02d-%02dT%02d:%02d:%02d",  //
                      x.Y, x.M, x.D,                         //
                      x.h, x.m, (int)(x.iJD / 1000 % 60));

  if (x.useSubsec) {
    int rem = x.iJD % 1000;
    if (rem) {
      sqlite3_str_appendchar(res, 1, '.');
      sqlite3_str_appendchar(res, 1, '0' + rem / 100);
      if ((rem %= 100)) {
        sqlite3_str_appendchar(res, 1, '0' + rem / 10);
        if ((rem %= 10)) {
          sqlite3_str_appendchar(res, 1, '0' + rem);
        }
      }
    }
  }

  if (x.tz) {
    sqlite3_str_appendf(res, "%+03d:%02d", x.tz / 60, abs(x.tz) % 60);
  } else {
    sqlite3_str_appendchar(res, 1, 'Z');
  }

  int rc = sqlite3_str_errcode(res);
  if (rc) {
    sqlite3_result_error_code(context, rc);
    return;
  }

  int n = sqlite3_str_length(res);
  sqlite3_result_text(context, sqlite3_str_finish(res), n, sqlite3_free);
}

int sqlite3_time_init(sqlite3 *db, char **pzErrMsg,
                      const sqlite3_api_routines *pApi) {
  sqlite3_create_collation_v2(db, "time", SQLITE_UTF8, /*arg=*/NULL,
                              time_collation,
                              /*destroy=*/NULL);
  sqlite3_create_function_v2(
      db, "json_time", -1,
      SQLITE_UTF8 | SQLITE_DETERMINISTIC | SQLITE_INNOCUOUS, /*arg=*/NULL,
      json_time_func, /*step=*/NULL, /*final=*/NULL, /*destroy=*/NULL);
  return SQLITE_OK;
}