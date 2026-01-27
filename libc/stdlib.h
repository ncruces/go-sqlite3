#pragma once

#include <stddef.h>

void abort(void);

void free(void*);
void* malloc(size_t);
void* calloc(size_t, size_t);
void* realloc(void*, size_t);

void* aligned_alloc(size_t, size_t);

void qsort(void*, size_t, size_t, int (*)(const void*, const void*));
