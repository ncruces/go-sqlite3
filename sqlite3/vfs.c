#include <stdbool.h>
#include <stddef.h>
#include <time.h>

#include "include.h"
#include "sqlite3.h"

int go_localtime(struct tm *, sqlite3_int64);
int go_vfs_find(const char *zVfsName);

int go_randomness(sqlite3_vfs *, int nByte, char *zOut);
int go_sleep(sqlite3_vfs *, int microseconds);
int go_current_time_64(sqlite3_vfs *, sqlite3_int64 *);

int go_open(sqlite3_vfs *, sqlite3_filename zName, sqlite3_file *, int flags,
            int *pOutFlags, int *pOutVFS);
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

int go_shm_map(sqlite3_file *, int iPg, int pgsz, int, void volatile **);
int go_shm_lock(sqlite3_file *, int offset, int n, int flags);
int go_shm_unmap(sqlite3_file *, int deleteFlag);
void go_shm_barrier(sqlite3_file *);

static int go_open_wrapper(sqlite3_vfs *vfs, sqlite3_filename zName,
                           sqlite3_file *file, int flags, int *pOutFlags) {
  static const sqlite3_io_methods go_io[2] = {
      {
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
      },
      {
          .iVersion = 2,
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
          .xShmMap = go_shm_map,
          .xShmLock = go_shm_lock,
          .xShmBarrier = go_shm_barrier,
          .xShmUnmap = go_shm_unmap,
      }};
  int vfsID = 0;
  memset(file, 0, vfs->szOsFile);
  int rc = go_open(vfs, zName, file, flags, pOutFlags, &vfsID);
  if (rc) {
    return rc;
  }
  file->pMethods = &go_io[vfsID];
  return SQLITE_OK;
}

struct go_file {
  sqlite3_file base;
  go_handle handle;
};

int sqlite3_os_init() {
  static sqlite3_vfs os_vfs = {
      .iVersion = 2,
      .szOsFile = sizeof(struct go_file),
      .mxPathname = 1024,
      .zName = "os",

      .xOpen = go_open_wrapper,
      .xDelete = go_delete,
      .xAccess = go_access,
      .xFullPathname = go_full_pathname,

      .xRandomness = go_randomness,
      .xSleep = go_sleep,
      .xCurrentTimeInt64 = go_current_time_64,
  };
  return sqlite3_vfs_register(&os_vfs, /*default=*/true);
}

int localtime_s(struct tm *const pTm, time_t const *const pTime) {
  return go_localtime(pTm, (sqlite3_int64)*pTime);
}

sqlite3_vfs *sqlite3_vfs_find(const char *zVfsName) {
  if (zVfsName && go_vfs_find(zVfsName)) {
    static sqlite3_vfs *go_vfs_list;

    for (sqlite3_vfs *it = go_vfs_list; it; it = it->pNext) {
      if (!strcmp(zVfsName, it->zName)) {
        return it;
      }
    }

    for (sqlite3_vfs **ptr = &go_vfs_list; *ptr;) {
      sqlite3_vfs *it = *ptr;
      if (go_vfs_find(it->zName)) {
        ptr = &it->pNext;
      } else {
        *ptr = it->pNext;
        free(it);
      }
    }

    sqlite3_vfs *head = go_vfs_list;
    go_vfs_list = malloc(sizeof(sqlite3_vfs) + strlen(zVfsName) + 1);
    char *name = (char *)(go_vfs_list + 1);
    strcpy(name, zVfsName);
    *go_vfs_list = (sqlite3_vfs){
        .iVersion = 2,
        .szOsFile = sizeof(struct go_file),
        .mxPathname = 1024,
        .zName = name,
        .pNext = head,

        .xOpen = go_open_wrapper,
        .xDelete = go_delete,
        .xAccess = go_access,
        .xFullPathname = go_full_pathname,

        .xRandomness = go_randomness,
        .xSleep = go_sleep,
        .xCurrentTimeInt64 = go_current_time_64,
    };
    return go_vfs_list;
  }
  return sqlite3_vfs_find_orig(zVfsName);
}

static_assert(offsetof(sqlite3_vfs, zName) == 16, "Unexpected offset");
static_assert(offsetof(struct go_file, handle) == 4, "Unexpected offset");