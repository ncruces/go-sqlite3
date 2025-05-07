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
void *memcpy(void *__restrict dest, const void *__restrict src, size_t n) {
  return __builtin_memcpy(dest, src, n);
}

__attribute__((weak))
void *memmove(void *dest, const void *src, size_t n) {
  return __builtin_memmove(dest, src, n);
}

#endif  // __wasm_bulk_memory__

#ifdef __wasm_simd128__

// SIMD implementations of string.h functions.

__attribute__((weak))
int memcmp(const void *v1, const void *v2, size_t n) {
  // Baseline algorithm.
  if (n < sizeof(v128_t)) {
    const unsigned char *u1 = (unsigned char *)v1;
    const unsigned char *u2 = (unsigned char *)v2;
    while (n--) {
      if (*u1 != *u2) return *u1 - *u2;
      u1++;
      u2++;
    }
    return 0;
  }

  // memcmp is allowed to read up to n bytes from each object.
  // Find the first different character in the objects.
  // Unaligned loads handle the case where the objects
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)v1;
  const v128_t *w2 = (v128_t *)v2;
  while (n) {
    const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(w1), wasm_v128_load(w2));
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(cmp)) {
      // Find the offset of the first zero bit (little-endian).
      size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
      const unsigned char *u1 = (unsigned char *)w1 + ctz;
      const unsigned char *u2 = (unsigned char *)w2 + ctz;
      // This may help the compiler if the function is inlined.
      __builtin_assume(*u1 - *u2 != 0);
      return *u1 - *u2;
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

__attribute__((weak))
void *memchr(const void *v, int c, size_t n) {
  // When n is zero, a function that locates a character finds no occurrence.
  // Otherwise, decrement n to ensure sub_overflow overflows
  // when n would go equal-to-or-below zero.
  if (n-- == 0) {
    return NULL;
  }

  // memchr must behave as if it reads characters sequentially
  // and stops as soon as a match is found.
  // Aligning ensures loads beyond the first match don't fail.
  uintptr_t align = (uintptr_t)v % sizeof(v128_t);
  const v128_t *w = (v128_t *)((char *)v - align);
  const v128_t wc = wasm_i8x16_splat(c);

  for (;;) {
    const v128_t cmp = wasm_i8x16_eq(*w, wc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to alignment (little-endian)
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
        return ctz <= n + align ? (char *)w + ctz : NULL;
      }
    }
    // Decrement n; if it overflows we're done.
    if (__builtin_sub_overflow(n, sizeof(v128_t) - align, &n)) {
      return NULL;
    }
    align = 0;
    w++;
  }
}

__attribute__((weak))
void *memrchr(const void *v, int c, size_t n) {
  // memrchr is allowed to read up to n bytes from the object.
  // Search backward for the last matching character.
  const v128_t *w = (v128_t *)((char *)v + n);
  const v128_t wc = wasm_i8x16_splat(c);
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(--w), wc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      size_t clz = __builtin_clz(wasm_i8x16_bitmask(cmp)) - 15;
      return (char *)(w + 1) - clz;
    }
  }

  // Baseline algorithm.
  const char *a = (char *)w;
  while (n--) {
    if (*(--a) == (char)c) return (char *)a;
  }
  return NULL;
}

__attribute__((weak))
size_t strlen(const char *s) {
  // strlen must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator don't fail.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (v128_t *)(s - align);

  for (;;) {
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(*w)) {
      const v128_t cmp = wasm_i8x16_eq(*w, (v128_t){});
      // Clear the bits corresponding to alignment (little-endian)
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

  // Unaligned loads handle the case where the strings
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)s1;
  const v128_t *w2 = (v128_t *)s2;
  while (w1 <= limit && w2 <= limit) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      // The strings may still be equal,
      // if the terminator is found before that difference.
      break;
    }
    // All characters are equal.
    // If any is a terminator the strings are equal.
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;
    }
    w1++;
    w2++;
  }

  // Baseline algorithm.
  const unsigned char *u1 = (unsigned char *)w1;
  const unsigned char *u2 = (unsigned char *)w2;
  for (;;) {
    if (*u1 != *u2) return *u1 - *u2;
    if (*u1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

static int __strcmp_s(const char *s1, const char *s2) {
  const unsigned char *u1 = (unsigned char *)s1;
  const unsigned char *u2 = (unsigned char *)s2;
  for (;;) {
    if (*u1 != *u2) return *u1 - *u2;
    if (*u1 == 0) break;
    u1++;
    u2++;
  }
  return 0;
}

__attribute__((weak, always_inline))
int strcmp(const char *s1, const char *s2) {
  // Skip the vector search when comparing against small literal strings.
  if (__builtin_constant_p(strlen(s2)) && strlen(s2) < sizeof(v128_t)) {
    return __strcmp_s(s1, s2);
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

  // Unaligned loads handle the case where the strings
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)s1;
  const v128_t *w2 = (v128_t *)s2;
  for (; w1 <= limit && w2 <= limit && n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      // The strings may still be equal,
      // if the terminator is found before that difference.
      break;
    }
    // All characters are equal.
    // If any is a terminator the strings are equal.
    if (!wasm_i8x16_all_true(wasm_v128_load(w1))) {
      return 0;
    }
    w1++;
    w2++;
  }

  // Baseline algorithm.
  const unsigned char *u1 = (unsigned char *)w1;
  const unsigned char *u2 = (unsigned char *)w2;
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
  // Aligning ensures loads beyond the first match don't fail.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (v128_t *)(s - align);
  const v128_t wc = wasm_i8x16_splat(c);

  for (;;) {
    const v128_t cmp = wasm_i8x16_eq(*w, (v128_t){}) | wasm_i8x16_eq(*w, wc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to alignment (little-endian)
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
  return *r == (char)c ? r : NULL;
}

__attribute__((weak, always_inline))
char *strrchr(const char *s, int c) {
  // For finding the terminator, strlen is faster.
  if (__builtin_constant_p(c) && (char)c == 0) {
    return (char *)s + strlen(s);
  }
  // This could also be implemented in a single pass using strchr,
  // advancing to the next match until no more matches are found.
  // That would be suboptimal with lots of consecutive matches.
  return (char *)memrchr(s, c, strlen(s) + 1);
}

__attribute__((weak))
size_t strspn(const char *s, const char *c) {
  // Set limit to the largest possible valid v128_t pointer.
  // Unsigned modular arithmetic gives the correct result
  // unless memory size is zero, in which case all pointers are invalid.
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  const v128_t *w = (v128_t *)s;
  const char *const a = s;

  if (!c[0]) return 0;
  if (!c[1]) {
    const v128_t wc = wasm_i8x16_splat(*c);
    while (w <= limit) {
      const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(w), wc);
      // Bitmask is slow on AArch64, all_true is much faster.
      if (!wasm_i8x16_all_true(cmp)) {
        size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
        return (char *)w + ctz - s;
      }
      w++;
    }

    // Baseline algorithm.
    for (s = (char *)w; *s == *c; s++);
    return s - a;
  }

  // http://0x80.pl/notesen/2018-10-18-simd-byte-lookup.html
  typedef unsigned char u8x16
      __attribute__((__vector_size__(16), __aligned__(16)));

  u8x16 bitmap07 = {};
  u8x16 bitmap8f = {};
  for (; *c; c++) {
    unsigned lo_nibble = *(unsigned char *)c % 16;
    unsigned hi_nibble = *(unsigned char *)c / 16;
    bitmap07[lo_nibble] |= 1 << (hi_nibble - 0);
    bitmap8f[lo_nibble] |= 1 << (hi_nibble - 8);
    // Terminator IS NOT on the bitmap.
  }

  for (; w <= limit; w++) {
    const v128_t lo_nibbles = wasm_v128_load(w) & wasm_u8x16_const_splat(0xf);
    const v128_t hi_nibbles = wasm_u8x16_shr(wasm_v128_load(w), 4);

    const v128_t bitmask_lookup =
        wasm_u8x16_const(1, 2, 4, 8, 16, 32, 64, 128,  //
                         1, 2, 4, 8, 16, 32, 64, 128);

    const v128_t bitmask = wasm_i8x16_swizzle(bitmask_lookup, hi_nibbles);
    const v128_t bitsets = wasm_v128_bitselect(
        wasm_i8x16_swizzle(bitmap07, lo_nibbles),
        wasm_i8x16_swizzle(bitmap8f, lo_nibbles),
        wasm_i8x16_lt(hi_nibbles, wasm_u8x16_const_splat(8)));

    const v128_t cmp = wasm_i8x16_eq(bitsets & bitmask, bitmask);
    if (!wasm_i8x16_all_true(cmp)) {
      size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
      return (char *)w + ctz - s;
    }
  }

  // Baseline algorithm.
  for (s = (char *)w;; s++) {
    const unsigned lo_nibble = *(unsigned char *)s & 0xf;
    const unsigned hi_nibble = *(unsigned char *)s >> 4;
    const unsigned bitmask = 1 << (hi_nibble & 0x7);
    const unsigned bitset =
        hi_nibble < 8 ? bitmap07[lo_nibble] : bitmap8f[lo_nibble];
    if ((bitset & bitmask) == 0) return s - a;
  }
}

__attribute__((weak))
size_t strcspn(const char *s, const char *c) {
  if (!c[0] || !c[1]) return __strchrnul(s, *c) - s;

  // Set limit to the largest possible valid v128_t pointer.
  // Unsigned modular arithmetic gives the correct result
  // unless memory size is zero, in which case all pointers are invalid.
  const v128_t *const limit =
      (v128_t *)(__builtin_wasm_memory_size(0) * PAGESIZE) - 1;

  const v128_t *w = (v128_t *)s;
  const char *const a = s;

  // http://0x80.pl/notesen/2018-10-18-simd-byte-lookup.html
  typedef unsigned char u8x16
      __attribute__((__vector_size__(16), __aligned__(16)));

  u8x16 bitmap07 = {};
  u8x16 bitmap8f = {};
  for (;; c++) {
    unsigned lo_nibble = *(unsigned char *)c % 16;
    unsigned hi_nibble = *(unsigned char *)c / 16;
    bitmap07[lo_nibble] |= 1 << (hi_nibble - 0);
    bitmap8f[lo_nibble] |= 1 << (hi_nibble - 8);
    if (!*c) break;  // Terminator IS on the bitmap.
  }

  for (; w <= limit; w++) {
    const v128_t lo_nibbles = wasm_v128_load(w) & wasm_u8x16_const_splat(0xf);
    const v128_t hi_nibbles = wasm_u8x16_shr(wasm_v128_load(w), 4);

    const v128_t bitmask_lookup =
        wasm_u8x16_const(1, 2, 4, 8, 16, 32, 64, 128,  //
                         1, 2, 4, 8, 16, 32, 64, 128);

    const v128_t bitmask = wasm_i8x16_swizzle(bitmask_lookup, hi_nibbles);
    const v128_t bitsets = wasm_v128_bitselect(
        wasm_i8x16_swizzle(bitmap07, lo_nibbles),
        wasm_i8x16_swizzle(bitmap8f, lo_nibbles),
        wasm_i8x16_lt(hi_nibbles, wasm_u8x16_const_splat(8)));

    const v128_t cmp = wasm_i8x16_eq(bitsets & bitmask, bitmask);
    if (wasm_v128_any_true(cmp)) {
      size_t ctz = __builtin_ctz(wasm_i8x16_bitmask(cmp));
      return (char *)w + ctz - s;
    }
  }

  // Baseline algorithm.
  for (s = (char *)w;; s++) {
    const unsigned lo_nibble = *(unsigned char *)s & 0xf;
    const unsigned hi_nibble = *(unsigned char *)s >> 4;
    const unsigned bitmask = 1 << (hi_nibble & 0x7);
    const unsigned bitset =
        hi_nibble < 8 ? bitmap07[lo_nibble] : bitmap8f[lo_nibble];
    if (bitset & bitmask) return s - a;
  }
}

// Given the above SIMD implementations,
// these are best implemented as
// small wrappers over those functions.

// Simple wrappers already in musl:
//  - mempcpy
//  - strcat
//  - strdup
//  - strndup
//  - strnlen
//  - strpbrk
//  - strsep
//  - strtok

__attribute__((weak))
void *memccpy(void *__restrict dest, const void *__restrict src, int c, size_t n) {
  void *memchr(const void *v, int c, size_t n);
  const void *m = memchr(src, c, n);
  if (m != NULL) {
    n = (char *)m - (char *)src + 1;
    m = (char *)dest + n;
  }
  memcpy(dest, src, n);
  return (void *)m;
}

__attribute__((weak))
char *strncat(char *__restrict dest, const char *__restrict src, size_t n) {
  size_t strnlen(const char *s, size_t n);
  size_t dlen = strlen(dest);
  size_t slen = strnlen(src, n);
  memcpy(dest + dlen, src, slen);
  dest[dlen + slen] = 0;
  return dest;
}

static char *__stpcpy(char *__restrict dest, const char *__restrict src) {
  size_t slen = strlen(src);
  memcpy(dest, src, slen + 1);
  return dest + slen;
}

static char *__stpncpy(char *__restrict dest, const char *__restrict src, size_t n) {
  size_t strnlen(const char *s, size_t n);
  size_t slen = strnlen(src, n);
  memcpy(dest, src, slen);
  memset(dest + slen, 0, n - slen);
  return dest + slen;
}

__attribute__((weak, always_inline))
char *stpcpy(char *__restrict dest, const char *__restrict src) {
  return __stpcpy(dest, src);
}

char *strcpy(char *__restrict dest, const char *__restrict src) {
  __stpcpy(dest, src);
  return dest;
}

__attribute__((weak, always_inline))
char *stpncpy(char *__restrict dest, const char *__restrict src, size_t n) {
  return __stpncpy(dest, src, n);
}

__attribute__((weak, always_inline))
char *strncpy(char *__restrict dest, const char *__restrict src, size_t n) {
  __stpncpy(dest, src, n);
  return dest;
}

#endif  // __wasm_simd128__

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // _WASM_SIMD128_STRING_H