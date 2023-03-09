#include <string.h>

#include "sqlite3.h"

static int time_collation(void *pArg, int nKey1, const void *pKey1, int nKey2,
                          const void *pKey2) {
  // If keys are of different length, and both terminated by a Z,
  // ignore the Z for collation purposes.
  if (nKey1 && nKey2 && nKey1 != nKey2) {
    const char *pK1 = (const char *)pKey1;
    const char *pK2 = (const char *)pKey2;
    if (pK1[nKey1 - 1] == 'Z' && pK2[nKey2 - 1] == 'Z') {
      nKey1--;
      nKey2--;
    }
  }

  int n = nKey1 < nKey2 ? nKey1 : nKey2;
  int rc = memcmp(pKey1, pKey2, n);
  if (rc == 0) {
    rc = nKey1 - nKey2;
  }
  return rc;
}

int sqlite3_time_collation(sqlite3 *db) {
  return sqlite3_create_collation_v2(db, "TIME", SQLITE_UTF8, 0, time_collation,
                                     0);
}