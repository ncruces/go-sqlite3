// A simple bump allocator that never frees memory.
// Takes over the initial heap, then grows it as needed.
// Assumes that new memory is zero-initialized,
// and that the heap base is 16 byte aligned.
// It allocates in 16 byte chunks and keeps no size metadata.

#include <stdlib.h>

#define PAGESIZE 65536

extern char __heap_base[];
extern char __heap_end[];

static void* __arena_beg = __heap_base;
static void* __arena_end = __heap_end;

__attribute__((always_inline)) void free(void*) {}

void* malloc(size_t size) {
  if (size == 0) return NULL;
  size = __builtin_align_up(size, 16);

  size_t avail = __arena_end - __arena_beg;
  if (size > avail) {
    size_t npages = (size - avail + PAGESIZE - 1) / PAGESIZE;
    size_t old = __builtin_wasm_memory_grow(0, npages);
    if (old == SIZE_MAX) return NULL;
    __arena_end += npages * PAGESIZE;
  }

  void* res = __arena_beg;
  __arena_beg += size;
  return res;
}

void* calloc(size_t nelem, size_t elsize) { return malloc(nelem * elsize); }

void* aligned_alloc(size_t align, size_t size) {
  // Ensure non-zero power-of-two.
  if (align <= 0 || (align & (align - 1))) return NULL;
  align = (align - (size_t)__arena_beg) & (align - 1);
  return malloc(size + align) + align;
}

void* realloc(void* ptr, size_t size) {
  if (ptr == NULL) return malloc(size);
  // No need to move the first chunk.
  if (size <= 16) return ptr;

  size_t copy = __arena_beg - ptr;
  void* res = malloc(size);
  if (copy > size) copy = size;
  __builtin_memcpy(res, ptr, copy);
  return res;
}
