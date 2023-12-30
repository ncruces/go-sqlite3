// Amalgamation
#include "sqlite3.c"
// Extensions
#include "ext/anycollseq.c"
#include "ext/base64.c"
#include "ext/decimal.c"
#include "ext/ieee754.c"
#include "ext/regexp.c"
#include "ext/series.c"
#include "ext/uint.c"
#include "ext/uuid.c"
// Bindings
#include "func.c"
#include "log.c"
#include "pointer.c"
#include "progress.c"
#include "time.c"
#include "vfs.c"
#include "vtab.c"

sqlite3_destructor_type malloc_destructor = &free;

__attribute__((constructor)) void init() {
  sqlite3_initialize();
  sqlite3_auto_extension((void (*)(void))sqlite3_base_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_decimal_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_ieee_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_regexp_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_series_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_uint_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_uuid_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_time_init);
}