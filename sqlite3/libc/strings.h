#include_next <strings.h>  // the system strings.h

#ifndef _WASM_SIMD128_STRINGS_H
#define _WASM_SIMD128_STRINGS_H

#include <ctype.h>
#include <stdint.h>
#include <wasm_simd128.h>
#include <__macro_PAGESIZE.h>

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_simd128__

#ifdef __OPTIMIZE_SIZE__

// bcmp is the same as memcmp but only compares for equality.
int bcmp(const void *v1, const void *v2, size_t n);

#else  // __OPTIMIZE_SIZE__

__attribute__((weak))
int bcmp(const void *v1, const void *v2, size_t n) {
  // Scalar algorithm.
  if (n < sizeof(v128_t)) {
    const unsigned char *u1 = (unsigned char *)v1;
    const unsigned char *u2 = (unsigned char *)v2;
    while (n--) {
      if (*u1 != *u2) return 1;
      u1++;
      u2++;
    }
    return 0;
  }

  // bcmp is allowed to read up to n bytes from each object.
  // Unaligned loads handle the case where the objects
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)v1;
  const v128_t *w2 = (v128_t *)v2;
  while (n) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      return 1;
    }
    // This makes n a multiple of sizeof(v128_t)
    // for every iteration except the first.
    size_t align = (n - 1) % sizeof(v128_t) + 1;
    w1 = (v128_t *)((char *)w1 + align);
    w2 = (v128_t *)((char *)w2 + align);
    n -= align;
  }
  return 0;
}

#endif  // __OPTIMIZE_SIZE__

__attribute__((always_inline))
static v128_t __tolower8x16(v128_t v) {
  __i8x16 i = v;
  i = i + wasm_i8x16_splat(INT8_MAX - ('Z'));
  i = i > wasm_i8x16_splat(INT8_MAX - ('Z' - 'A' + 1));
  i = i & wasm_i8x16_splat('a' - 'A');
  return v | i;
}

static int __strcasecmp_s(const char *s1, const char *s2) {
  // Scalar algorithm.
  const unsigned char *u1 = (unsigned char *)s1;
  const unsigned char *u2 = (unsigned char *)s2;
  for (;;) {
    int c1 = tolower(*u1);
    int c2 = tolower(*u2);
    if (c1 != c2) return c1 - c2;
    if (c1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

static int __strcasecmp(const char *s1, const char *s2) {
  // How many bytes can be read before pointers go out of bounds.
  size_t N = __builtin_wasm_memory_size(0) * PAGESIZE -  //
             (size_t)(s1 > s2 ? s1 : s2);

  // Unaligned loads handle the case where the strings
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)s1;
  const v128_t *w2 = (v128_t *)s2;
  for (; N >= sizeof(v128_t); N -= sizeof(v128_t)) {
    v128_t v1 = __tolower8x16(wasm_v128_load(w1));
    v128_t v2 = __tolower8x16(wasm_v128_load(w2));

    // Find any single bit difference.
    if (wasm_v128_any_true(v1 ^ v2)) {
      // The terminator may come before the difference.
      break;
    }
    // We know all characters are equal.
    // If any is a terminator the strings are equal.
    if (!wasm_i8x16_all_true(v1)) {
      return 0;
    }
    w1++;
    w2++;
  }

  return __strcasecmp_s((char *)w1, (char *)w2);
}

__attribute__((weak))
int strcasecmp(const char *s1, const char *s2) {
  // Skip the vector search when comparing against small literal strings.
  if (__builtin_constant_p(strlen(s2)) && strlen(s2) < sizeof(v128_t)) {
    return __strcasecmp_s(s1, s2);
  }
  return __strcasecmp(s1, s2);
}

__attribute__((weak))
int strncasecmp(const char *s1, const char *s2, size_t n) {
  // How many bytes can be read before pointers go out of bounds.
  size_t N = __builtin_wasm_memory_size(0) * PAGESIZE -  //
             (size_t)(s1 > s2 ? s1 : s2);
  if (n > N) n = N;

  // Unaligned loads handle the case where the strings
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)s1;
  const v128_t *w2 = (v128_t *)s2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    v128_t v1 = __tolower8x16(wasm_v128_load(w1));
    v128_t v2 = __tolower8x16(wasm_v128_load(w2));

    // Find any single bit difference.
    if (wasm_v128_any_true(v1 ^ v2)) {
      // The terminator may come before the difference.
      break;
    }
    // We know all characters are equal.
    // If any is a terminator the strings are equal.
    if (!wasm_i8x16_all_true(v1)) {
      return 0;
    }
    w1++;
    w2++;
  }

  // Scalar algorithm.
  const unsigned char *u1 = (unsigned char *)w1;
  const unsigned char *u2 = (unsigned char *)w2;
  while (n--) {
    int c1 = tolower(*u1);
    int c2 = tolower(*u2);
    if (c1 != c2) return c1 - c2;
    if (c1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

#endif  // __wasm_simd128__

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // _WASM_SIMD128_STRINGS_H