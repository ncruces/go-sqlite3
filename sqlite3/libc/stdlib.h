#pragma once

#include <stddef.h>

__attribute__((noreturn)) void abort(void);

void free(void*);
__attribute__((malloc)) void* malloc(size_t);
__attribute__((malloc)) void* calloc(size_t, size_t);
__attribute__((malloc)) void* aligned_alloc(size_t, size_t);
void* realloc(void*, size_t);

void qsort(void*, size_t, size_t, int (*)(const void*, const void*));
