// Amalgamation
#include "sqlite3.c"
// Extensions
#include "ext/anycollseq.c"
#include "ext/base64.c"
#include "ext/decimal.c"
#include "ext/ieee754.c"
#include "ext/regexp.c"
#include "ext/series.c"
#include "ext/spellfix.c"
#include "ext/uint.c"
// Bindings
#include "func.c"
#include "hooks.c"
#include "pointer.c"
#include "stmt.c"
#include "text.c"
#include "time.c"
#include "vfs.c"
#include "vtab.c"
// Libc
#include "libc.c"
#include "malloc_sbrk.c"

void _initialize() {
  try_init_allocator();
  sqlite3_initialize();
  sqlite3_auto_extension((void (*)(void))sqlite3_base_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_decimal_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_ieee_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_regexp_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_series_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_spellfix_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_uint_init);
  sqlite3_auto_extension((void (*)(void))sqlite3_time_init);
}
