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

int sqlite3_time_collation(sqlite3 *db) {
  return sqlite3_create_collation(db, "TIME", SQLITE_UTF8, 0, time_collation);
}