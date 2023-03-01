#include <time.h>

#include "sqlite3.h"

int os_localtime(sqlite3_int64, struct tm *);

int os_randomness(sqlite3_vfs *, int nByte, char *zOut);
int os_sleep(sqlite3_vfs *, int microseconds);
int os_current_time(sqlite3_vfs *, double *);
int os_current_time_64(sqlite3_vfs *, sqlite3_int64 *);

int os_open(sqlite3_vfs *, sqlite3_filename zName, sqlite3_file *, int flags,
            int *pOutFlags);
int os_delete(sqlite3_vfs *, const char *zName, int syncDir);
int os_access(sqlite3_vfs *, const char *zName, int flags, int *pResOut);
int os_full_pathname(sqlite3_vfs *, const char *zName, int nOut, char *zOut);

struct os_file {
  sqlite3_file base;
  int id;
  int lock;
};

int os_close(sqlite3_file *);
int os_read(sqlite3_file *, void *, int iAmt, sqlite3_int64 iOfst);
int os_write(sqlite3_file *, const void *, int iAmt, sqlite3_int64 iOfst);
int os_truncate(sqlite3_file *, sqlite3_int64 size);
int os_sync(sqlite3_file *, int flags);
int os_file_size(sqlite3_file *, sqlite3_int64 *pSize);
int os_file_control(sqlite3_file *pFile, int op, void *pArg);

int os_lock(sqlite3_file *pFile, int eLock);
int os_unlock(sqlite3_file *pFile, int eLock);
int os_check_reserved_lock(sqlite3_file *pFile, int *pResOut);

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
  return os_localtime((sqlite3_int64)*pTime, pTm);
}

static int os_open_w(sqlite3_vfs *vfs, sqlite3_filename zName,
                     sqlite3_file *file, int flags, int *pOutFlags) {
  static const sqlite3_io_methods os_io = {
      .iVersion = 1,
      .xClose = os_close,
      .xRead = os_read,
      .xWrite = os_write,
      .xTruncate = os_truncate,
      .xSync = os_sync,
      .xFileSize = os_file_size,
      .xLock = os_lock,
      .xUnlock = os_unlock,
      .xCheckReservedLock = os_check_reserved_lock,
      .xFileControl = no_file_control,
      .xDeviceCharacteristics = no_device_characteristics,
  };
  int rc = os_open(vfs, zName, file, flags, pOutFlags);
  file->pMethods = (char)rc == SQLITE_OK ? &os_io : NULL;
  return rc;
}

sqlite3_vfs *os_vfs() {
  static sqlite3_vfs os_vfs = {
      .iVersion = 2,
      .szOsFile = sizeof(struct os_file),
      .mxPathname = 512,
      .zName = "os",

      .xOpen = os_open_w,
      .xDelete = os_delete,
      .xAccess = os_access,
      .xFullPathname = os_full_pathname,

      .xRandomness = os_randomness,
      .xSleep = os_sleep,
      .xCurrentTime = os_current_time,
      .xCurrentTimeInt64 = os_current_time_64,
  };
  return &os_vfs;
}
