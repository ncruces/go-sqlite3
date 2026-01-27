#include <stdint.h>
#include <stdlib.h>
#include <string.h>

extern char __heap_base[];
extern char __heap_end[];

static void* __arena_beg = __heap_base;
static void* __arena_end = __heap_end;

static void __arena_grow(size_t size) {
  size_t avail = __arena_end - __arena_beg;
  if (size > avail) {
    size_t npages = (size - avail + 0xfffful) >> 16;
    size_t old = __builtin_wasm_memory_grow(0, npages);
    if (old == ~0ul) __builtin_trap();
    __arena_end += npages << 16;
  }
}

__attribute__((always_inline)) void free(void*) {}

__attribute__((malloc)) void* malloc(size_t size) {
  if (size == 0) return NULL;
  size = (size + 15) & ~15;
  void* res = __arena_beg;
  __arena_grow(size);
  __arena_beg += size;
  return res;
}

__attribute__((malloc)) void* calloc(size_t nelem, size_t elsize) {
  return malloc(nelem * elsize);
}

__attribute__((malloc)) void* aligned_alloc(size_t align, size_t size) {
  if ((align & -align) != align) __builtin_trap();
  align = (align - ((uintptr_t)__arena_beg & (align - 1))) & (align - 1);
  return malloc(size + align) + align;
}

void* realloc(void* ptr, size_t size) {
  size_t copy = __arena_beg - ptr;
  void* res = malloc(size);
  if (ptr != NULL && res != NULL) {
    if (size < copy) copy = size;
    memcpy(res, ptr, copy);
  }
  return res;
}

__attribute__((noreturn)) void abort(void) { __builtin_trap(); }

// Shellsort with Gonnet & Baeza-Yates gap sequence.
// Simple, no recursion, doesn't use the C stack.
// Clang auto-vectorizes the inner loop.

void qsort(void* base, size_t nel, size_t width,
           int (*comp)(const void*, const void*)) {
  // If nel is zero, we're required to do nothing.
  // If it's one, the array is already sorted.
  size_t wnel = width * nel;
  size_t gap = nel;
  while (gap > 1) {
    // Use 64-bit unsigned arithmetic to avoid intermediate overflow.
    // Absent overflow, gap will be strictly less than its previous value.
    // Once it is one or zero, set it to one: do a final pass, and stop.
    gap = (5ull * gap - 1) / 11;
    if (gap == 0) gap = 1;

    // It'd be undefined behavior for wnel to overflow a size_t;
    // or if width is zero: the base pointer would be invalid.
    // Since gap is stricly less than nel, we can assume
    // wgap is strictly less than wnel.
    size_t wgap = width * gap;
    __builtin_assume(wgap < wnel);
    for (size_t i = wgap; i < wnel; i += width) {
      // Even without overflow flags, the overflow builtin helps the compiler.
      for (size_t j = i; !__builtin_sub_overflow(j, wgap, &j);) {
        char* a = j + (char*)base;
        char* b = a + wgap;
        if (comp(a, b) <= 0) break;

        // This well known loop is automatically vectorized.
        size_t s = width;
        do {
          char tmp = *a;
          *a++ = *b;
          *b++ = tmp;
        } while (--s);
      }
    }
  }
}
