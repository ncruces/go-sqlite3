#include <stddef.h>

#include "main.c"
#include "os.c"
#include "qsort.c"
#include "sqlite3.c"
#include "time.c"

sqlite3_destructor_type malloc_destructor = &free;
size_t sqlite3_interrupt_offset = offsetof(sqlite3, u1.isInterrupted);