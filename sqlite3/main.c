#include <stdbool.h>
#include <stdlib.h>
#include <time.h>

#include "sqlite3.h"

int main() {
  int rc = sqlite3_initialize();
  if (rc != SQLITE_OK) return 1;
}

int go_localtime(sqlite3_int64, struct tm *);

int go_randomness(sqlite3_vfs *, int nByte, char *zOut);
int go_sleep(sqlite3_vfs *, int microseconds);
int go_current_time(sqlite3_vfs *, double *);
int go_current_time_64(sqlite3_vfs *, sqlite3_int64 *);

int go_open(sqlite3_vfs *, sqlite3_filename zName, sqlite3_file *, int flags,
            int *pOutFlags);
int go_delete(sqlite3_vfs *, const char *zName, int syncDir);
int go_access(sqlite3_vfs *, const char *zName, int flags, int *pResOut);
int go_full_pathname(sqlite3_vfs *, const char *zName, int nOut, char *zOut);

struct go_file {
  sqlite3_file base;
  int id;
  int eLock;
};

int go_close(sqlite3_file *);
int go_read(sqlite3_file *, void *, int iAmt, sqlite3_int64 iOfst);
int go_write(sqlite3_file *, const void *, int iAmt, sqlite3_int64 iOfst);
int go_truncate(sqlite3_file *, sqlite3_int64 size);
int go_sync(sqlite3_file *, int flags);
int go_file_size(sqlite3_file *, sqlite3_int64 *pSize);
int go_file_control(sqlite3_file *pFile, int op, void *pArg);

int go_lock(sqlite3_file *pFile, int eLock);
int go_unlock(sqlite3_file *pFile, int eLock);
int go_check_reserved_lock(sqlite3_file *pFile, int *pResOut);

static int no_lock(sqlite3_file *pFile, int eLock) { return SQLITE_OK; }
static int no_unlock(sqlite3_file *pFile, int eLock) { return SQLITE_OK; }
static int no_check_reserved_lock(sqlite3_file *pFile, int *pResOut) {
  *pResOut = 0;
  return SQLITE_OK;
}

static int no_file_control(sqlite3_file *pFile, int op, void *pArg) {
  return SQLITE_NOTFOUND;
}
static int no_sector_size(sqlite3_file *pFile) { return 0; }
static int no_device_characteristics(sqlite3_file *pFile) { return 0; }

int localtime_s(struct tm *const pTm, time_t const *const pTime) {
  return go_localtime((sqlite3_int64)*pTime, pTm);
}

static int go_open_c(sqlite3_vfs *vfs, sqlite3_filename zName,
                     sqlite3_file *file, int flags, int *pOutFlags) {
  static const sqlite3_io_methods go_io = {
      .iVersion = 1,
      .xClose = go_close,
      .xRead = go_read,
      .xWrite = go_write,
      .xTruncate = go_truncate,
      .xSync = go_sync,
      .xFileSize = go_file_size,
      .xLock = go_lock,
      .xUnlock = go_unlock,
      .xCheckReservedLock = go_check_reserved_lock,
      .xFileControl = no_file_control,
      .xSectorSize = no_sector_size,
      .xDeviceCharacteristics = no_device_characteristics,
  };
  int rc = go_open(vfs, zName, file, flags, pOutFlags);
  file->pMethods = (char)rc == SQLITE_OK ? &go_io : NULL;
  return rc;
}

int sqlite3_os_init() {
  static sqlite3_vfs go_vfs = {
      .iVersion = 2,
      .szOsFile = sizeof(struct go_file),
      .mxPathname = 512,
      .zName = "go",

      .xOpen = go_open_c,
      .xDelete = go_delete,
      .xAccess = go_access,
      .xFullPathname = go_full_pathname,

      .xRandomness = go_randomness,
      .xSleep = go_sleep,
      .xCurrentTime = go_current_time,
      .xCurrentTimeInt64 = go_current_time_64,
  };
  return sqlite3_vfs_register(&go_vfs, /*default=*/true);
}

sqlite3_destructor_type malloc_destructor = &free;