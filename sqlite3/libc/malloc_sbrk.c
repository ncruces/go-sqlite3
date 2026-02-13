#define PAGESIZE 65536

void* sbrk(intptr_t increment) {
  if (increment == 0) return (void*)(__builtin_wasm_memory_size(0) * PAGESIZE);
  if (increment < 0 || increment % PAGESIZE != 0) abort();

  size_t old = __builtin_wasm_memory_grow(0, (size_t)increment / PAGESIZE);
  if (old == SIZE_MAX) return (void*)old;
  return (void*)(old * PAGESIZE);
}

#define LACKS_ERRNO_H
#define LACKS_FCNTL_H
#define LACKS_SCHED_H
#define LACKS_STRINGS_H
#define LACKS_SYS_MMAN_H
#define LACKS_SYS_PARAM_H
#define LACKS_SYS_TYPES_H
#define LACKS_TIME_H
#define LACKS_UNISTD_H

#define HAVE_MMAP 0
#define MALLOC_ALIGNMENT 16
#define MALLOC_FAILURE_ACTION
#define MORECORE_CANNOT_TRIM 1
#define NO_MALLINFO 1
#define NO_MALLOC_STATS 1
#define USE_BUILTIN_FFS 1
#define USE_LOCKS 0

#define ffs(i) (__builtin_ffs(i))

#define ENOMEM 7   // SQLITE_NOMEM
#define EINVAL 21  // SQLITE_MISUSE

#pragma clang diagnostic ignored "-Weverything"

#include "malloc.c"

extern char __heap_base[];
extern char __heap_end[];

// Initialize dlmalloc to be able to use the memory between
// __heap_base and __heap_end.
static void try_init_allocator(void) {
  if (is_initialized(gm)) __builtin_trap();
  ensure_initialization();

  size_t heap_size = __heap_end - __heap_base;
  if (heap_size <= MIN_CHUNK_SIZE + TOP_FOOT_SIZE + MALLOC_ALIGNMENT) return;

  gm->least_addr = __heap_base;
  gm->seg.base = __heap_base;
  gm->seg.size = heap_size;
  gm->footprint = heap_size;
  gm->max_footprint = heap_size;
  gm->magic = mparams.magic;
  gm->release_checks = MAX_RELEASE_CHECK_RATE;

  init_bins(gm);
  init_top(gm, (mchunkptr)__heap_base, heap_size - TOP_FOOT_SIZE);
}

__attribute__((alias("memalign"))) void* aligned_alloc(size_t align,
                                                       size_t size);
