#include <stdlib.h>

#include "sqlite3.h"

void sqlite3_result_text_go(sqlite3_context* ctx, const char* zData,
                            sqlite3_uint64 nData) {
  sqlite3_result_text64(ctx, zData, nData, &sqlite3_free, SQLITE_UTF8);
}

void sqlite3_result_blob_go(sqlite3_context* ctx, const void* zData,
                            sqlite3_uint64 nData) {
  sqlite3_result_blob64(ctx, zData, nData, &sqlite3_free);
}