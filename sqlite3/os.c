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
  int id;
  char lock;
  char psow;
  char syncDir;
  int lockTimeout;
};

static_assert(offsetof(struct os_file, id) == 4, "Unexpected offset");
static_assert(offsetof(struct os_file, lock) == 8, "Unexpected offset");
static_assert(offsetof(struct os_file, psow) == 9, "Unexpected offset");
static_assert(offsetof(struct os_file, syncDir) == 10, "Unexpected offset");
static_assert(offsetof(struct os_file, lockTimeout) == 12, "Unexpected offset");

int os_close(sqlite3_file *);
int os_read(sqlite3_file *, void *, int iAmt, sqlite3_int64 iOfst);
int os_write(sqlite3_file *, const void *, int iAmt, sqlite3_int64 iOfst);
int os_truncate(sqlite3_file *, sqlite3_int64 size);
int os_sync(sqlite3_file *, int flags);
int os_file_size(sqlite3_file *, sqlite3_int64 *pSize);
int os_file_control(sqlite3_file *, int op, void *pArg);

int os_lock(sqlite3_file *, int eLock);
int os_unlock(sqlite3_file *, int eLock);
int os_check_reserved_lock(sqlite3_file *, int *pResOut);

static int os_file_control_w(sqlite3_file *file, int op, void *pArg) {
  struct os_file *pFile = (struct os_file *)file;
  switch (op) {
    case SQLITE_FCNTL_VFSNAME: {
      *(char **)pArg = sqlite3_mprintf("%s", "os");
      return SQLITE_OK;
    }
    case SQLITE_FCNTL_LOCKSTATE: {
      *(int *)pArg = pFile->lock;
      return SQLITE_OK;
    }
    case SQLITE_FCNTL_LOCK_TIMEOUT: {
      int iOld = pFile->lockTimeout;
      pFile->lockTimeout = *(int *)pArg;
      *(int *)pArg = iOld;
      return SQLITE_OK;
    }
    case SQLITE_FCNTL_POWERSAFE_OVERWRITE: {
      if (*(int *)pArg < 0) {
        *(int *)pArg = pFile->psow;
      } else {
        pFile->psow = *(int *)pArg;
      }
      return SQLITE_OK;
    }
    case SQLITE_FCNTL_SIZE_HINT:
    case SQLITE_FCNTL_HAS_MOVED:
      return os_file_control(file, op, pArg);
  }
  // Consider also implementing these opcodes (in use by SQLite):
  //  SQLITE_FCNTL_BUSYHANDLER
  //  SQLITE_FCNTL_COMMIT_PHASETWO
  //  SQLITE_FCNTL_PDB
  //  SQLITE_FCNTL_PRAGMA
  //  SQLITE_FCNTL_SYNC
  return SQLITE_NOTFOUND;
}

static int os_sector_size(sqlite3_file *file) {
  return SQLITE_DEFAULT_SECTOR_SIZE;
}

static int os_device_characteristics(sqlite3_file *file) {
  struct os_file *pFile = (struct os_file *)file;
  return pFile->psow ? SQLITE_IOCAP_POWERSAFE_OVERWRITE : 0;
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

  struct os_file *pFile = (struct os_file *)file;
  pFile->base.pMethods = &os_io;
  if (flags & SQLITE_OPEN_MAIN_DB) {
    pFile->psow =
        sqlite3_uri_boolean(zName, "psow", SQLITE_POWERSAFE_OVERWRITE);
  }
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
