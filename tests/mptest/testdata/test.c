#define unlink dont_unlink

#include "mptest.c"

int dont_unlink(const char *pathname) { return 0; }