// A simple bump allocator that never frees memory.
// Takes over the initial heap, then grows it as needed.
// Assumes new memory is zero-initialized.
// Assumes heap base is 16 byte aligned, and allocates 16 byte chunks.

#include <stdlib.h>

extern char __heap_base[];
extern char __heap_end[];

static void* __arena_beg = __heap_base;
static void* __arena_end = __heap_end;

__attribute__((always_inline)) void free(void*) {}

void* malloc(size_t size) {
  if (size == 0) return NULL;
  size = (size + 15) & ~15;

  size_t avail = __arena_end - __arena_beg;
  if (size > avail) {
    size_t npages = (size - avail + 65535) >> 16;
    size_t old = __builtin_wasm_memory_grow(0, npages);
    if (old == SIZE_MAX) return NULL;
    __arena_end += npages << 16;
  }

  void* res = __arena_beg;
  __arena_beg += size;
  return res;
}

void* calloc(size_t nelem, size_t elsize) { return malloc(nelem * elsize); }

void* aligned_alloc(size_t align, size_t size) {
  if (align == 0 || (align & (align - 1))) return NULL;
  align = (align - (size_t)__arena_beg) & (align - 1);
  return malloc(size + align) + align;
}

void* realloc(void* ptr, size_t size) {
  if (size <= 16 && ptr != NULL) return ptr;
  size_t copy = __arena_beg - ptr;
  void* res = malloc(size);
  if (ptr != NULL) {
    if (size < copy) copy = size;
    __builtin_memcpy(res, ptr, copy);
  }
  return res;
}
