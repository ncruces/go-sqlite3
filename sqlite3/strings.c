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

  const unsigned char *u1 = (const void *)w1;
  const unsigned char *u2 = (const void *)w2;
  while (n--) {
    if (*u1 != *u2) return *u1 - *u2;
    u1++;
    u2++;
  }
  return 0;
}

int strncmp(const char *c1, const char *c2, size_t n) {
  const v128_t *w1 = (const void *)c1;
  const v128_t *w2 = (const void *)c2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;  // *w1 != *w2
    }
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;  // *w1 == *w2 and they have a NUL
    }
    w1++;
    w2++;
  }

  c1 = (const void *)w1;
  c2 = (const void *)w2;
  while (n-- && *c1 == *c2) {
    if (n == 0 || *c1 == 0) return 0;
    c1++;
    c2++;
  }
  return *(unsigned char *)c1 - *(unsigned char *)c2;
}

#endif

#define ONES (~(uintmax_t)(0) / UCHAR_MAX)
#define HIGHS (ONES * (UCHAR_MAX / 2 + 1))
#define HASZERO(x) ((x) - (typeof(x))(ONES) & ~(x) & (typeof(x))(HIGHS))
#define UNALIGNED(x) ((uintptr_t)(x) & (sizeof(*x) - 1))

int strcmp(const char *c1, const char *c2) {
  typedef uintptr_t __attribute__((__may_alias__)) word;

  const word *w1 = (const void *)c1;
  const word *w2 = (const void *)c2;
  if (!(UNALIGNED(w1) | UNALIGNED(w2))) {
    while (*w1 == *w2) {
      if (HASZERO(*w1)) return 0;
      w1++;
      w2++;
    }
    c1 = (const void *)w1;
    c2 = (const void *)w2;
  }

  while (*c1 == *c2 && *c1) {
    c1++;
    c2++;
  }
  return *(unsigned char *)c1 - *(unsigned char *)c2;
}

#undef UNALIGNED
#undef HASZERO
#undef HIGHS
#undef ONES