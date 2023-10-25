#include <stddef.h>

#include "sqlite3.h"

int go_progress(void *);

void sqlite3_progress_handler_go(sqlite3 *db, int n) {
  sqlite3_progress_handler(db, n, go_progress, NULL);
}
