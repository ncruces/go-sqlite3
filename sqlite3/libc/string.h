#include_next <string.h>  // the system string.h

#ifndef _WASM_SIMD128_STRING_H
#define _WASM_SIMD128_STRING_H

#include <ctype.h>
#include <stdint.h>
#include <strings.h>
#include <wasm_simd128.h>
#include <__macro_PAGESIZE.h>

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_bulk_memory__

// Use the builtins if compiled with bulk memory operations.
// Clang will intrinsify using SIMD for small, constant N.

__attribute__((weak, always_inline))
void *memset(void *dest, int c, size_t n) {
  return __builtin_memset(dest, c, n);
}

__attribute__((weak, always_inline))
void *memcpy(void *__restrict dest, const void *__restrict src, size_t n) {
  return __builtin_memcpy(dest, src, n);
}

__attribute__((weak, always_inline))
void *memmove(void *dest, const void *src, size_t n) {
  return __builtin_memmove(dest, src, n);
}

#endif  // __wasm_bulk_memory__

#ifdef __wasm_simd128__

__attribute__((weak))
int memcmp(const void *vl, const void *vr, size_t n) {
  // Scalar algorithm.
  if (n < sizeof(v128_t)) {
    const unsigned char *u1 = (unsigned char *)vl;
    const unsigned char *u2 = (unsigned char *)vr;
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
  const v128_t *v1 = (v128_t *)vl;
  const v128_t *v2 = (v128_t *)vr;
  while (n) {
    const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(v1), wasm_v128_load(v2));
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(cmp)) {
      // Find the offset of the first zero bit (little-endian).
      size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
      const unsigned char *u1 = (unsigned char *)v1 + ctz;
      const unsigned char *u2 = (unsigned char *)v2 + ctz;
      // This may help the compiler if the function is inlined.
      __builtin_assume(*u1 - *u2 != 0);
      return *u1 - *u2;
    }
    // This makes n a multiple of sizeof(v128_t)
    // for every iteration except the first.
    size_t align = (n - 1) % sizeof(v128_t) + 1;
    v1 = (v128_t *)((char *)v1 + align);
    v2 = (v128_t *)((char *)v2 + align);
    n -= align;
  }
  return 0;
}

__attribute__((weak))
void *memchr(const void *s, int c, size_t n) {
  // When n is zero, a function that locates a character finds no occurrence.
  // Otherwise, decrement n to ensure sub_overflow overflows
  // when n would go equal-to-or-below zero.
  if (!n--) {
    return NULL;
  }

  // memchr must behave as if it reads characters sequentially
  // and stops as soon as a match is found.
  // Aligning ensures loads beyond the first match are safe.
  // Casting through uintptr_t makes this implementation-defined,
  // rather than undefined behavior.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *v = (v128_t *)((uintptr_t)s - align);
  const v128_t vc = wasm_i8x16_splat(c);

  for (;;) {
    const v128_t cmp = wasm_i8x16_eq(*v, vc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to align (little-endian)
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless align cleared them.
      // Knowing this helps the compiler if it unrolls the loop.
      __builtin_assume(mask || align);
      // If the mask became zero because of align,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        // That's a match, unless it is beyond the end of the object.
        // Recall that we decremented n, so less-than-or-equal-to is correct.
        size_t ctz = __builtin_ctz(mask);
        return ctz - align <= n ? (char *)v + ctz : NULL;
      }
    }
    // Decrement n; if it overflows we're done.
    if (__builtin_sub_overflow(n, sizeof(v128_t) - align, &n)) {
      return NULL;
    }
    align = 0;
    v++;
  }
}

__attribute__((weak))
void *memrchr(const void *s, int c, size_t n) {
  // memrchr is allowed to read up to n bytes from the object.
  // Search backward for the last matching character.
  const v128_t *v = (v128_t *)((char *)s + n);
  const v128_t vc = wasm_i8x16_splat(c);
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(--v), vc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Find the offset of the last one bit (little-endian).
      // The leading 16 bits of the bitmask are always zero,
      // and to be ignored.
      size_t clz = __builtin_clz(wasm_i8x16_bitmask(cmp)) - 16;
      return (char *)(v + 1) - (clz + 1);
    }
  }

  // Scalar algorithm.
  const char *a = (char *)v;
  while (n--) {
    if (*(--a) == (char)c) return (char *)a;
  }
  return NULL;
}

__attribute__((weak))
size_t strlen(const char *s) {
  // strlen must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator are safe.
  // Casting through uintptr_t makes this implementation-defined,
  // rather than undefined behavior.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *v = (v128_t *)((uintptr_t)s - align);

  for (;;) {
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(*v)) {
      const v128_t cmp = wasm_i8x16_eq(*v, (v128_t){});
      // Clear the bits corresponding to align (little-endian)
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless align cleared them.
      // Knowing this helps the compiler if it unrolls the loop.
      __builtin_assume(mask || align);
      // If the mask became zero because of align,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        return (char *)v - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    v++;
  }
}

static char *__strchrnul(const char *s, int c) {
  // strchrnul must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator are safe.
  // Casting through uintptr_t makes this implementation-defined,
  // rather than undefined behavior.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *v = (v128_t *)((uintptr_t)s - align);
  const v128_t vc = wasm_i8x16_splat(c);

  for (;;) {
    const v128_t cmp = wasm_i8x16_eq(*v, (v128_t){}) | wasm_i8x16_eq(*v, vc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to align (little-endian)
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless align cleared them.
      // Knowing this helps the compiler if it unrolls the loop.
      __builtin_assume(mask || align);
      // If the mask became zero because of align,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        return (char *)v + __builtin_ctz(mask);
      }
    }
    align = 0;
    v++;
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

// SIMDized check which bytes are in a set (Geoff Langdale)
// http://0x80.pl/notesen/2018-10-18-simd-byte-lookup.html

typedef struct {
  __u8x16 l;
  __u8x16 h;
} __wasm_v128_bitmap256_t;

__attribute__((always_inline))
static void __wasm_v128_setbit(__wasm_v128_bitmap256_t *bitmap, int i) {
  uint8_t hi_nibble = (uint8_t)i >> 4;
  uint8_t lo_nibble = (uint8_t)i & 0xf;
  bitmap->l[lo_nibble] |= 1 << (hi_nibble - 0);
  bitmap->h[lo_nibble] |= 1 << (hi_nibble - 8);
}

#ifndef __wasm_relaxed_simd__

#define wasm_i8x16_relaxed_swizzle wasm_i8x16_swizzle

#endif  // __wasm_relaxed_simd__

__attribute__((always_inline))
static v128_t __wasm_v128_chkbits(__wasm_v128_bitmap256_t bitmap, v128_t v) {
  v128_t indices_0_7 = v & wasm_u8x16_const_splat(0x8f);
  v128_t indices_8_15 = (v & wasm_u8x16_const_splat(0x80)) ^ indices_0_7;

  v128_t row_0_7 = wasm_i8x16_swizzle(bitmap.l, indices_0_7);
  v128_t row_8_15 = wasm_i8x16_swizzle(bitmap.h, indices_8_15);

  v128_t bitsets = row_0_7 | row_8_15;

  v128_t hi_nibbles = wasm_u8x16_shr(v, 4);
  v128_t bitmask_lookup = wasm_u8x16_const(1, 2, 4, 8, 16, 32, 64, 128,  //
                                           1, 2, 4, 8, 16, 32, 64, 128);
  v128_t bitmask = wasm_i8x16_relaxed_swizzle(bitmask_lookup, hi_nibbles);

  return wasm_i8x16_eq(bitsets & bitmask, bitmask);
}

#undef wasm_i8x16_relaxed_swizzle

__attribute__((weak))
size_t strspn(const char *s, const char *c) {
  // strspn must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator are safe.
  // Casting through uintptr_t makes this implementation-defined,
  // rather than undefined behavior.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *v = (v128_t *)((uintptr_t)s - align);

  if (!c[0]) return 0;
  if (!c[1]) {
    const v128_t vc = wasm_i8x16_splat(*c);
    for (;;) {
      const v128_t cmp = wasm_i8x16_eq(*v, vc);
      // Bitmask is slow on AArch64, all_true is much faster.
      if (!wasm_i8x16_all_true(cmp)) {
        // Clear the bits corresponding to align (little-endian)
        // so we can count trailing zeros.
        int mask = (uint16_t)~wasm_i8x16_bitmask(cmp) >> align << align;
        // At least one bit will be set, unless align cleared them.
        // Knowing this helps the compiler if it unrolls the loop.
        __builtin_assume(mask || align);
        // If the mask became zero because of align,
        // it's as if we didn't find anything.
        if (mask) {
          // Find the offset of the first one bit (little-endian).
          return (char *)v - s + __builtin_ctz(mask);
        }
      }
      align = 0;
      v++;
    }
  }

  __wasm_v128_bitmap256_t bitmap = {};

  for (; *c; c++) {
    // Terminator IS NOT on the bitmap.
    __wasm_v128_setbit(&bitmap, *c);
  }

  for (;;) {
    const v128_t cmp = __wasm_v128_chkbits(bitmap, *v);
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(cmp)) {
      // Clear the bits corresponding to align (little-endian)
      // so we can count trailing zeros.
      int mask = (uint16_t)~wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless align cleared them.
      // Knowing this helps the compiler if it unrolls the loop.
      __builtin_assume(mask || align);
      // If the mask became zero because of align,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        return (char *)v - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    v++;
  }
}

__attribute__((weak))
size_t strcspn(const char *s, const char *c) {
  if (!c[0] || !c[1]) return __strchrnul(s, *c) - s;

  // strcspn must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator are safe.
  // Casting through uintptr_t makes this implementation-defined,
  // rather than undefined behavior.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *v = (v128_t *)((uintptr_t)s - align);

  __wasm_v128_bitmap256_t bitmap = {};

  do {
    // Terminator IS on the bitmap.
    __wasm_v128_setbit(&bitmap, *c);
  } while (*c++);

  for (;;) {
    const v128_t cmp = __wasm_v128_chkbits(bitmap, *v);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Clear the bits corresponding to align (little-endian)
      // so we can count trailing zeros.
      int mask = wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless align cleared them.
      // Knowing this helps the compiler if it unrolls the loop.
      __builtin_assume(mask || align);
      // If the mask became zero because of align,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        return (char *)v - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    v++;
  }
}

// Given the above SIMD implementations,
// these are best implemented as
// small wrappers over those functions.

// Simple wrappers already in musl:
//  - mempcpy
//  - strcat
//  - strlcat
//  - strdup
//  - strndup
//  - strnlen
//  - strpbrk
//  - strsep
//  - strtok

__attribute__((weak))
void *memccpy(void *__restrict dest, const void *__restrict src, int c,
              size_t n) {
  const void *m = memchr(src, c, n);
  if (m != NULL) {
    n = (char *)m - (char *)src + 1;
    m = (char *)dest + n;
  }
  memcpy(dest, src, n);
  return (void *)m;
}

__attribute__((weak))
size_t strlcpy(char *__restrict dest, const char *__restrict src, size_t n) {
  size_t slen = strlen(src);
  if (n--) {
    if (n > slen) n = slen;
    memcpy(dest, src, n);
    dest[n] = 0;
  }
  return slen;
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

static char *__stpncpy(char *__restrict dest, const char *__restrict src,
                       size_t n) {
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

__attribute__((weak, always_inline))
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