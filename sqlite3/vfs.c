#include <time.h>

#include "sqlite3.h"

int go_localtime(struct tm *, sqlite3_int64);

int go_randomness(sqlite3_vfs *, int nByte, char *zOut);
int go_sleep(sqlite3_vfs *, int microseconds);
int go_current_time(sqlite3_vfs *, double *);
int go_current_time_64(sqlite3_vfs *, sqlite3_int64 *);

int go_open(sqlite3_vfs *, sqlite3_filename zName, sqlite3_file *, int flags,
            int *pOutFlags);
int go_delete(sqlite3_vfs *, const char *zName, int syncDir);
int go_access(sqlite3_vfs *, const char *zName, int flags, int *pResOut);
int go_full_pathname(sqlite3_vfs *, const char *zName, int nOut, char *zOut);

int go_close(sqlite3_file *);
int go_read(sqlite3_file *, void *, int iAmt, sqlite3_int64 iOfst);
int go_write(sqlite3_file *, const void *, int iAmt, sqlite3_int64 iOfst);
int go_truncate(sqlite3_file *, sqlite3_int64 size);
int go_sync(sqlite3_file *, int flags);
int go_file_size(sqlite3_file *, sqlite3_int64 *pSize);
int go_file_control(sqlite3_file *, int op, void *pArg);
int go_sector_size(sqlite3_file *file);
int go_device_characteristics(sqlite3_file *file);

int go_lock(sqlite3_file *, int eLock);
int go_unlock(sqlite3_file *, int eLock);
int go_check_reserved_lock(sqlite3_file *, int *pResOut);

static int go_open_wrapper(sqlite3_vfs *vfs, sqlite3_filename zName,
                           sqlite3_file *file, int flags, int *pOutFlags) {
  static const sqlite3_io_methods os_io = {
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
      .xFileControl = go_file_control,
      .xSectorSize = go_sector_size,
      .xDeviceCharacteristics = go_device_characteristics,
  };
  memset(file, 0, vfs->szOsFile);
  int rc = go_open(vfs, zName, file, flags, pOutFlags);
  if (rc) {
    return rc;
  }
  file->pMethods = &os_io;
  return SQLITE_OK;
}

struct go_file {
  sqlite3_file base;
  int handle;
};

static_assert(offsetof(struct go_file, handle) == 4, "Unexpected offset");

sqlite3_vfs *os_vfs() {
  static sqlite3_vfs os_vfs = {
      .iVersion = 2,
      .szOsFile = sizeof(struct go_file),
      .mxPathname = 512,
      .zName = "os",

      .xOpen = go_open_wrapper,
      .xDelete = go_delete,
      .xAccess = go_access,
      .xFullPathname = go_full_pathname,

      .xRandomness = go_randomness,
      .xSleep = go_sleep,
      .xCurrentTime = go_current_time,
      .xCurrentTimeInt64 = go_current_time_64,
  };
  return &os_vfs;
}

int localtime_s(struct tm *const pTm, time_t const *const pTime) {
  return go_localtime(pTm, (sqlite3_int64)*pTime);
}
