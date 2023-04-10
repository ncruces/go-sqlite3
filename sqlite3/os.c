#include <time.h>

#include "sqlite3.h"

int os_localtime(struct tm *, sqlite3_int64);

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
  int handle;
};

static_assert(offsetof(struct os_file, handle) == 4, "Unexpected offset");

int os_close(sqlite3_file *);
int os_read(sqlite3_file *, void *, int iAmt, sqlite3_int64 iOfst);
int os_write(sqlite3_file *, const void *, int iAmt, sqlite3_int64 iOfst);
int os_truncate(sqlite3_file *, sqlite3_int64 size);
int os_sync(sqlite3_file *, int flags);
int os_file_size(sqlite3_file *, sqlite3_int64 *pSize);
int os_file_control(sqlite3_file *, int op, void *pArg);
int os_sector_size(sqlite3_file *file);
int os_device_characteristics(sqlite3_file *file);

int os_lock(sqlite3_file *, int eLock);
int os_unlock(sqlite3_file *, int eLock);
int os_check_reserved_lock(sqlite3_file *, int *pResOut);

static int os_file_control_w(sqlite3_file *file, int op, void *pArg) {
  struct os_file *pFile = (struct os_file *)file;
  if (op == SQLITE_FCNTL_VFSNAME) {
      *(char **)pArg = sqlite3_mprintf("%s", "os");
      return SQLITE_OK;
  }
  return os_file_control(file, op, pArg);
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
      .xFileControl = os_file_control_w,
      .xSectorSize = os_sector_size,
      .xDeviceCharacteristics = os_device_characteristics,
  };
  memset(file, 0, sizeof(struct os_file));
  int rc = os_open(vfs, zName, file, flags, pOutFlags);
  if (rc) {
    return rc;
  }

  file->pMethods = &os_io;
  return SQLITE_OK;
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

int localtime_s(struct tm *const pTm, time_t const *const pTime) {
  return os_localtime(pTm, (sqlite3_int64)*pTime);
}
