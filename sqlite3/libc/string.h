#ifndef _WASM_SIMD128_STRING_H
#define _WASM_SIMD128_STRING_H

#include <limits.h>
#include <stddef.h>
#include <stdint.h>
#include <wasm_simd128.h>
#include <__macro_PAGESIZE.h>

#include_next <string.h>  // the system string.h

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_bulk_memory__

// Use the builtins if compiled with bulk memory operations.
// Clang will intrinsify using SIMD for small, constant N.
// For everything else, this helps inlining.

__attribute__((weak))
void *memset(void *dest, int c, size_t n) {
  return __builtin_memset(dest, c, n);
}

__attribute__((weak))
void *memcpy(void *restrict dest, const void *restrict src, size_t n) {
  return __builtin_memcpy(dest, src, n);
}

__attribute__((weak))
void *memmove(void *dest, const void *src, size_t n) {
  return __builtin_memmove(dest, src, n);
}

#endif  // __wasm_bulk_memory__

#ifdef __wasm_simd128__

// SIMD versions of some string.h functions.
//
// These assume aligned v128_t loads can't fail,
// and so can't unaligned loads up to the last
// aligned address less than memory size.
//
// These also assume unaligned access is not painfully slow,
// but that bitmask extraction is really slow on AArch64.

__attribute__((weak))
int memcmp(const void *v1, const void *v2, size_t n) {
  // memcmp can read up to n bytes from each object.
  // Use unaligned loads to handle the case where
  // the objects have mismatching alignments.
  const v128_t *w1 = v1;
  const v128_t *w2 = v2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;
    }
    w1++;
    w2++;
  }

  // Continue byte-by-byte.
  const unsigned char *u1 = (void *)w1;
  const unsigned char *u2 = (void *)w2;
  while (n--) {
    if (*u1 != *u2) return *u1 - *u2;
    u1++;
    u2++;
  }
  return 0;
}

__attribute__((weak))
void *memchr(const void *v, int c, size_t n) {
  // When n is zero, a function that locates a character finds no occurrence.
  // Otherwise, decrement n to ensure __builtin_sub_overflow "overflows"
  // when n would go equal-to-or-below zero.
  if (n-- == 0) {
    return NULL;
  }

  // memchr must behave as if it reads characters sequentially
  // and stops as soon as a match is found.
  // Aligning ensures loads can't fail.
  uintptr_t align = (uintptr_t)v % sizeof(v128_t);
  const v128_t *w = (void *)(v - align);
  const v128_t wc = wasm_i8x16_splat(c);

  while (true) {
    const v128_t cmp = wasm_i8x16_eq(*w, wc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to alignment
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless we cleared them.
      // Knowing this helps the compiler. 
      __builtin_assume(mask || align);
      // If the mask is zero because of alignment,
      // it's as if we didn't find anything.
      if (mask) {
        // We found a match, unless it is beyond the end of the object.
        // Recall that we decremented n, so less-than-or-equal-to is correct.
        size_t ctz = __builtin_ctz(mask);
        return ctz <= n + align ? (void *)w + ctz : NULL;
      }
    }
    // Decrement n; if it "overflows" we're done.
    if (__builtin_sub_overflow(n, sizeof(v128_t) - align, &n)) {
      return NULL;
    }
    align = 0;
    w++;
  }
}

__attribute__((weak))
size_t strlen(const char *s) {
  // strlen must stop as soon as it finds the terminator.
  // Aligning ensures loads can't fail.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (void *)(s - align);

  while (true) {
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(*w)) {
      const v128_t cmp = wasm_i8x16_eq(*w, (v128_t){});
      // Clear the bits corresponding to alignment
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless we cleared them.
      // Knowing this helps the compiler. 
      __builtin_assume(mask || align);
      if (mask) {
        return (char *)w - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

static int __strcmp(const char *s1, const char *s2) {
  // Set limit to the largest possible valid v128_t pointer.
  // Unsigned modular arithmetic gives the correct result
  // unless memory size is zero, in which case all pointers are invalid.
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  // Use unaligned loads to handle the case where
  // the strings have mismatching alignments.
  const v128_t *w1 = (void *)s1;
  const v128_t *w2 = (void *)s2;
  while (w1 <= limit && w2 <= limit) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;
    }
    // All bytes are equal.
    // If any byte is zero (on both strings) the strings are equal.
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;
    }
    w1++;
    w2++;
  }

  // Continue byte-by-byte.
  const unsigned char *u1 = (void *)w1;
  const unsigned char *u2 = (void *)w2;
  while (true) {
    if (*u1 != *u2) return *u1 - *u2;
    if (*u1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

__attribute__((weak, always_inline))
int strcmp(const char *s1, const char *s2) {
  // Use strncmp when comparing against literal strings.
  // If the literal is small, the vector search will be skipped.
  if (__builtin_constant_p(strlen(s2))) {
    return strncmp(s1, s2, strlen(s2));
  }
  return __strcmp(s1, s2);
}

__attribute__((weak))
int strncmp(const char *s1, const char *s2, size_t n) {
  // Set limit to the largest possible valid v128_t pointer.
  // Unsigned modular arithmetic gives the correct result
  // unless memory size is zero, in which case all pointers are invalid.
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  // Use unaligned loads to handle the case where
  // the strings have mismatching alignments.
  const v128_t *w1 = (void *)s1;
  const v128_t *w2 = (void *)s2;
  for (; w1 <= limit && w2 <= limit && n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      break;
    }
    // All bytes are equal.
    // If any byte is zero (on both strings) the strings are equal.
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;
    }
    w1++;
    w2++;
  }

  // Continue byte-by-byte.
  const unsigned char *u1 = (void *)w1;
  const unsigned char *u2 = (void *)w2;
  while (n--) {
    if (*u1 != *u2) return *u1 - *u2;
    if (*u1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

static char *__strchrnul(const char *s, int c) {
  // strchrnul must stop as soon as a match is found.
  // Aligning ensures loads can't fail.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (void *)(s - align);
  const v128_t wc = wasm_i8x16_splat(c);

  while (true) {
    const v128_t cmp = wasm_i8x16_eq(*w, (v128_t){}) | wasm_i8x16_eq(*w, wc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to alignment
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless we cleared them.
      // Knowing this helps the compiler. 
      __builtin_assume(mask || align);
      if (mask) {
        return (char *)w + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

__attribute__((weak, always_inline))
char *strchrnul(const char *s, int c) {
  // For finding the terminator, strlen is faster.
  if (__builtin_constant_p(c) && (char)c == 0) {
    return (char *)s + strlen(s);
  }
  return __strchrnul(s, c);
}

__attribute__((weak, always_inline))
char *strchr(const char *s, int c) {
  // For finding the terminator, strlen is faster.
  if (__builtin_constant_p(c) && (char)c == 0) {
    return (char *)s + strlen(s);
  }
  char *r = __strchrnul(s, c);
  return *(char *)r == (char)c ? r : NULL;
}

__attribute__((weak))
size_t strspn(const char *s, const char *c) {
#ifndef _REENTRANT
  static // Avoid the stack for builds without threads.
#endif
  char byteset[UCHAR_MAX + 1];
  const char *const a = s;

  if (!c[0]) return 0;
  if (!c[1]) {
    // Set limit to the largest possible valid v128_t pointer.
    // Unsigned modular arithmetic gives the correct result
    // unless memory size is zero, in which case all pointers are invalid.
    const v128_t *const limit =
        (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

    const v128_t *w = (void *)s;
    const v128_t wc = wasm_i8x16_splat(*c);
    while (w <= limit) {
      if (!wasm_i8x16_all_true(wasm_i8x16_eq(wasm_v128_load(w), wc))) {
        break;
      }
      w++;
    }

    s = (void *)w;
    while (*s == *c) s++;
    return s - a;
  }

#if !__OPTIMIZE__ || __OPTIMIZE_SIZE__

  // Unoptimized version.
  memset(byteset, 0, sizeof(byteset));
  while (*c && (byteset[*(unsigned char *)c] = 1)) c++;
  while (byteset[*(unsigned char *)s]) s++;

#else

  // This is faster than memset.
  volatile v128_t *w = (void *)byteset;
  #pragma unroll
  for (size_t i = sizeof(byteset) / sizeof(v128_t); i--;) w[i] = (v128_t){};
  static_assert(sizeof(byteset) % sizeof(v128_t) == 0);

  // Keeping byteset[0] = 0 avoids the other loop having to test for it.
  while (*c && (byteset[*(unsigned char *)c] = 1)) c++;
  #pragma unroll 4
  while (byteset[*(unsigned char *)s]) s++;

#endif

  return s - a;
}

__attribute__((weak))
size_t strcspn(const char *s, const char *c) {
#ifndef _REENTRANT
  static // Avoid the stack for builds without threads.
#endif
  char byteset[UCHAR_MAX + 1];
  const char *const a = s;

  if (!c[0] || !c[1]) return __strchrnul(s, *c) - s;

#if !__OPTIMIZE__ || __OPTIMIZE_SIZE__

  // Unoptimized version.
  memset(byteset, 0, sizeof(byteset));
  while ((byteset[*(unsigned char *)c] = 1) && *c) c++;
  while (!byteset[*(unsigned char *)s]) s++;

#else

  // This is faster than memset.
  volatile v128_t *w = (void *)byteset;
  #pragma unroll
  for (size_t i = sizeof(byteset) / sizeof(v128_t); i--;) w[i] = (v128_t){};
  static_assert(sizeof(byteset) % sizeof(v128_t) == 0);

  // Setting byteset[0] = 1 avoids the other loop having to test for it.
  while ((byteset[*(unsigned char *)c] = 1) && *c) c++;
  #pragma unroll 4
  while (!byteset[*(unsigned char *)s]) s++;

#endif

  return s - a;
}

#endif  // __wasm_simd128__

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // _WASM_SIMD128_STRING_H