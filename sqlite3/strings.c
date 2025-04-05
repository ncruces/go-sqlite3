#include <__macro_PAGESIZE.h>
#include <limits.h>
#include <stddef.h>
#include <stdint.h>
#include <wasm_simd128.h>

#ifdef __wasm_bulk_memory__

void *memset(void *dest, int c, size_t n) {
  return __builtin_memset(dest, c, n);
}

void *memcpy(void *restrict dest, const void *restrict src, size_t n) {
  return __builtin_memcpy(dest, src, n);
}

void *memmove(void *dest, const void *src, size_t n) {
  return __builtin_memmove(dest, src, n);
}

#endif

#ifdef __wasm_simd128__

int memcmp(const void *v1, const void *v2, size_t n) {
  const v128_t *w1 = v1;
  const v128_t *w2 = v2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;  // *w1 != *w2
    }
    w1++;
    w2++;
  }

  const uint8_t *u1 = (void *)w1;
  const uint8_t *u2 = (void *)w2;
  while (n--) {
    if (*u1 != *u2) return *u1 - *u2;
    u1++;
    u2++;
  }
  return 0;
}

size_t strlen(const char *s) {
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  const v128_t *w = (void *)s;
  while (w <= limit) {
    if (!wasm_i8x16_all_true(wasm_v128_load(w))) {
      break;  // *w has a NUL
    }
    w++;
  }

  const char *ss = (void *)w;
  while (true) {
    if (*ss == 0) break;
    ss++;
  }
  return ss - s;
}

int strcmp(const char *s1, const char *s2) {
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  const v128_t *w1 = (void *)s1;
  const v128_t *w2 = (void *)s2;
  while (w1 <= limit && w2 <= limit) {
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;  // *w1 != *w2
    }
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;  // *w1 == *w2 and have a NUL
    }
    w1++;
    w2++;
  }

  const uint8_t *u1 = (void *)w1;
  const uint8_t *u2 = (void *)w2;
  while (true) {
    if (*u1 != *u2) return *u1 - *u2;
    if (*u1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

int strncmp(const char *s1, const char *s2, size_t n) {
  const v128_t *w1 = (void *)s1;
  const v128_t *w2 = (void *)s2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;  // *w1 != *w2
    }
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;  // *w1 == *w2 and have a NUL
    }
    w1++;
    w2++;
  }

  const uint8_t *u1 = (void *)w1;
  const uint8_t *u2 = (void *)w2;
  while (n--) {
    if (*u1 != *u2) return *u1 - *u2;
    if (*u1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

#endif
