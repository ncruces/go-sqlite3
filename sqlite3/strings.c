#include <stddef.h>
#include <stdint.h>
#include <wasm_simd128.h>
#include <__macro_PAGESIZE.h>

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
      break;
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

void *memchr(const void *v, int c, size_t n) {
  uintptr_t align = (uintptr_t)v % sizeof(v128_t);
  const v128_t *w = (void *)(v - align);
  const v128_t wc = wasm_i8x16_splat(c);

  while (true) {
    const v128_t cmp = wasm_i8x16_eq(*w, wc);
    if (wasm_v128_any_true(cmp)) {
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      __builtin_assume(mask || align);
      if (mask) {
        return (void *)w + __builtin_ctz(mask);
      }
    }
    if (__builtin_sub_overflow(n, sizeof(v128_t) - align, &n)) {
      return NULL;
    }
    align = 0;
    w++;
  }
}

size_t strlen(const char *s) {
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (void *)(s - align);

  while (true) {
    if (!wasm_i8x16_all_true(*w)) {
      const v128_t cmp = wasm_i8x16_eq(*w, (v128_t){});
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      __builtin_assume(mask || align);
      if (mask) {
        return (char *)w - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

int strcmp(const char *s1, const char *s2) {
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  const v128_t *w1 = (void *)s1;
  const v128_t *w2 = (void *)s2;
  while (w1 <= limit && w2 <= limit) {
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;
    }
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;
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
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  const v128_t *w1 = (void *)s1;
  const v128_t *w2 = (void *)s2;
  for (; w1 <= limit && w2 <= limit && n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;
    }
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;
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

char *strchrnul(const char *s, int c) {
  if (__builtin_constant_p(c) && (char)c == 0) {
    return (char *)s + strlen(s);
  }

  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (void *)(s - align);
  const v128_t wc = wasm_i8x16_splat(c);

  while (true) {
    const v128_t cmp = wasm_i8x16_eq(*w, (v128_t){}) | wasm_i8x16_eq(*w, wc);
    if (wasm_v128_any_true(cmp)) {
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      __builtin_assume(mask || align);
      if (mask) {
        return (char *)w + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

char *strchr(const char *s, int c) {
  char *r = strchrnul(s, c);
  return *(char *)r == (char)c ? r : NULL;
}

#endif

#define BITOP(a, b, op)                          \
  ((a)[(b) / (8 * sizeof(size_t))] op((size_t)1) \
   << ((b) % (8 * sizeof(size_t))))

size_t strspn(const char *s, const char *c) {
  if (!c[0]) return 0;

  const char *const a = s;
  size_t byteset[32 / sizeof(size_t)] = {0};

  for (; *c && BITOP(byteset, *(uint8_t *)c, |=); c++);
  for (; *s && BITOP(byteset, *(uint8_t *)s, &); s++);
  return s - a;
}

size_t strcspn(const char *s, const char *c) {
  if (!c[0] || !c[1]) return strchrnul(s, *c) - s;

  const char *const a = s;
  size_t byteset[32 / sizeof(size_t)] = {0};

  for (; *c && BITOP(byteset, *(uint8_t *)c, |=); c++);
  for (; *s && !BITOP(byteset, *(uint8_t *)s, &); s++);
  return s - a;
}

#undef BITOP
