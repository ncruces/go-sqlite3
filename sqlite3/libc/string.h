#ifndef _WASM_SIMD128_STRING_H
#define _WASM_SIMD128_STRING_H

#include <stddef.h>
#include <stdint.h>
#include <strings.h>
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
  if (!n--) {
    return NULL;
  }

  // memchr must behave as if it reads characters sequentially
  // and stops as soon as a match is found.
  // Aligning ensures loads beyond the first match are safe.
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
        // Find the offset of the first one bit (little-endian).
        // That's a match, unless it is beyond the end of the object.
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
      // Find the offset of the last one bit (little-endian).
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
  // Aligning ensures loads beyond the terminator are safe.
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
        // Find the offset of the first one bit (little-endian).
        return (char *)w - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

static int __strcmp(const char *s1, const char *s2) {
  // How many bytes can be read before pointers go out of bounds.
  size_t N = __builtin_wasm_memory_size(0) * PAGESIZE -  //
             (size_t)(s1 > s2 ? s1 : s2);

  // Unaligned loads handle the case where the strings
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)s1;
  const v128_t *w2 = (v128_t *)s2;
  for (; N >= sizeof(v128_t); N -= sizeof(v128_t)) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      // The terminator may come before the difference.
      break;
    }
    // We know all characters are equal.
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
  // How many bytes can be read before pointers go out of bounds.
  size_t N = __builtin_wasm_memory_size(0) * PAGESIZE -  //
             (size_t)(s1 > s2 ? s1 : s2);
  if (n > N) n = N;

  // Unaligned loads handle the case where the strings
  // have mismatching alignments.
  const v128_t *w1 = (v128_t *)s1;
  const v128_t *w2 = (v128_t *)s2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      // The terminator may come before the difference.
      break;
    }
    // We know all characters are equal.
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
  // Aligning ensures loads beyond the first match are safe.
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
        // Find the offset of the first one bit (little-endian).
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

// http://0x80.pl/notesen/2018-10-18-simd-byte-lookup.html

#define _WASM_SIMD128_BITMAP256_T                                    \
  struct {                                                           \
    uint8_t l __attribute__((__vector_size__(16), __aligned__(16))); \
    uint8_t h __attribute__((__vector_size__(16), __aligned__(16))); \
  }

#define _WASM_SIMD128_SETBIT(bitmap, i)            \
  ({                                               \
    uint8_t _c = (uint8_t)(i);                     \
    uint8_t _hi_nibble = _c >> 4;                  \
    uint8_t _lo_nibble = _c & 0xf;                 \
    bitmap.l[_lo_nibble] |= 1 << (_hi_nibble - 0); \
    bitmap.h[_lo_nibble] |= 1 << (_hi_nibble - 8); \
  })

#define _WASM_SIMD128_CHKBIT(bitmap, i)                                   \
  ({                                                                      \
    uint8_t _c = (uint8_t)(i);                                            \
    uint8_t _hi_nibble = _c >> 4;                                         \
    uint8_t _lo_nibble = _c & 0xf;                                        \
    uint8_t _bitmask = 1 << (_hi_nibble & 0x7);                           \
    uint8_t _bitset = (_hi_nibble < 8 ? bitmap.l : bitmap.h)[_lo_nibble]; \
    _bitmask & _bitset;                                                   \
  })

#define _WASM_SIMD128_CHKBITS(bitmap, v)                                    \
  ({                                                                        \
    v128_t _w = v;                                                          \
    v128_t _hi_nibbles = wasm_u8x16_shr(_w, 4);                             \
    v128_t _lo_nibbles = _w & wasm_u8x16_const_splat(0xf);                  \
                                                                            \
    v128_t _bitmask_lookup = wasm_u8x16_const(1, 2, 4, 8, 16, 32, 64, 128,  \
                                              1, 2, 4, 8, 16, 32, 64, 128); \
                                                                            \
    v128_t _bitmask = wasm_i8x16_swizzle(_bitmask_lookup, _hi_nibbles);     \
    v128_t _bitsets = wasm_v128_bitselect(                                  \
        wasm_i8x16_swizzle(bitmap.l, _lo_nibbles),                          \
        wasm_i8x16_swizzle(bitmap.h, _lo_nibbles),                          \
        wasm_i8x16_lt(_hi_nibbles, wasm_u8x16_const_splat(8)));             \
                                                                            \
    wasm_i8x16_eq(_bitsets & _bitmask, _bitmask);                           \
  })

__attribute__((weak))
size_t strspn(const char *s, const char *c) {
  // How many bytes can be read before the pointer goes out of bounds.
  size_t N = __builtin_wasm_memory_size(0) * PAGESIZE - (size_t)s;
  const v128_t *w = (v128_t *)s;
  const char *const a = s;

  if (!c[0]) return 0;
  if (!c[1]) {
    const v128_t wc = wasm_i8x16_splat(*c);
    for (; N >= sizeof(v128_t); N -= sizeof(v128_t)) {
      const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(w), wc);
      // Bitmask is slow on AArch64, all_true is much faster.
      if (!wasm_i8x16_all_true(cmp)) {
        // Find the offset of the first zero bit (little-endian).
        size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
        return (char *)w + ctz - s;
      }
      w++;
    }

    // Baseline algorithm.
    for (s = (char *)w; *s == *c; s++);
    return s - a;
  }

  _WASM_SIMD128_BITMAP256_T bitmap = {};

  for (; *c; c++) {
    _WASM_SIMD128_SETBIT(bitmap, *c);
    // Terminator IS NOT on the bitmap.
  }

  for (; N >= sizeof(v128_t); N -= sizeof(v128_t)) {
    const v128_t cmp = _WASM_SIMD128_CHKBITS(bitmap, wasm_v128_load(w));
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(cmp)) {
      // Find the offset of the first zero bit (little-endian).
      size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
      return (char *)w + ctz - s;
    }
    w++;
  }

  // Baseline algorithm.
  for (s = (char *)w; _WASM_SIMD128_CHKBIT(bitmap, *s); s++);
  return s - a;
}

__attribute__((weak))
size_t strcspn(const char *s, const char *c) {
  if (!c[0] || !c[1]) return __strchrnul(s, *c) - s;

  // How many bytes can be read before the pointer goes out of bounds.
  size_t N = __builtin_wasm_memory_size(0) * PAGESIZE - (size_t)s;
  const v128_t *w = (v128_t *)s;
  const char *const a = s;

  _WASM_SIMD128_BITMAP256_T bitmap = {};

  for (;;) {
    _WASM_SIMD128_SETBIT(bitmap, *c);
    // Terminator IS on the bitmap.
    if (!*c++) break;
  }

  for (; N >= sizeof(v128_t); N -= sizeof(v128_t)) {
    const v128_t cmp = _WASM_SIMD128_CHKBITS(bitmap, wasm_v128_load(w));
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Find the offset of the first one bit (little-endian).
      size_t ctz = __builtin_ctz(wasm_i8x16_bitmask(cmp));
      return (char *)w + ctz - s;
    }
    w++;
  }

  // Baseline algorithm.
  for (s = (char *)w; !_WASM_SIMD128_CHKBIT(bitmap, *s); s++);
  return s - a;
}

#undef _WASM_SIMD128_SETBIT
#undef _WASM_SIMD128_CHKBIT
#undef _WASM_SIMD128_CHKBITS
#undef _WASM_SIMD128_BITMAP256_T

static const char *__memmem_raita(const char *haystk, size_t sh,
                                  const char *needle, size_t sn,
                                  uint8_t bmbc[256]) {
  // https://www-igm.univ-mlv.fr/~lecroq/string/node22.html
  // http://0x80.pl/notesen/2016-11-28-simd-strfind.html

  // We've handled empty and single character needles.
  // The needle is not longer than the haystack.
  __builtin_assume(2 <= sn && sn <= sh);

  // Find the farthest character not equal to the first one.
  size_t i = sn - 1;
  while (i > 0 && needle[0] == needle[i]) i--;
  if (i == 0) i = sn - 1;

  const v128_t fst = wasm_i8x16_splat(needle[0]);
  const v128_t lst = wasm_i8x16_splat(needle[i]);

  // The last haystk offset for which loading blk_lst is safe.
  const char *H = (char *)(__builtin_wasm_memory_size(0) * PAGESIZE - i -
                           sizeof(v128_t));

  while (haystk <= H) {
    const v128_t blk_fst = wasm_v128_load((v128_t *)(haystk));
    const v128_t blk_lst = wasm_v128_load((v128_t *)(haystk + i));
    const v128_t eq_fst = wasm_i8x16_eq(fst, blk_fst);
    const v128_t eq_lst = wasm_i8x16_eq(lst, blk_lst);

    const v128_t cmp = eq_fst & eq_lst;
    if (wasm_v128_any_true(cmp)) {
      // The terminator may come before the match.
      if (sh == SIZE_MAX && !wasm_i8x16_all_true(blk_fst)) break;
      // Find the offset of the first one bit (little-endian).
      // Each iteration clears that bit, tries again.
      for (uint32_t mask = wasm_i8x16_bitmask(cmp); mask; mask &= mask - 1) {
        size_t ctz = __builtin_ctz(mask);
        if (!bcmp(haystk + ctz + 1, needle + 1, sn - 1)) {
          return haystk + ctz;
        }
      }
    }

    size_t skip = sizeof(v128_t);
    // Apply the bad-character rule to the last checked
    // character of the haystack.
    if (bmbc) skip += bmbc[(unsigned char)haystk[sn + 14]];
    if (sh != SIZE_MAX) {
      // Have we reached the end of the haystack?
      if (__builtin_sub_overflow(sh, skip, &sh)) return NULL;
      // Is the needle longer than the haystack?
      if (sn > sh) return NULL;
    } else if (!wasm_i8x16_all_true(blk_fst)) {
      // We found a terminator.
      return NULL;
    }
    haystk += skip;
  }

  // Baseline algorithm.
  for (size_t j = 0; j <= sh - sn; j++) {
    for (size_t i = 0;; i++) {
      if (sn == i) return haystk;
      if (sh == SIZE_MAX && !haystk[i]) return NULL;
      if (needle[i] != haystk[i]) break;
    }
    haystk++;
  }
  return NULL;
}

static const char *__memmem(const char *haystk, size_t sh,  //
                            const char *needle, size_t sn) {
  // Is Boyer-Moore's bad-character rule useful?
  if (sn < sizeof(v128_t) || sh - sn < sizeof(v128_t)) {
    return __memmem_raita(haystk, sh, needle, sn, NULL);
  }

  // https://www-igm.univ-mlv.fr/~lecroq/string/node14.html

  // We've handled empty and single character needles.
  // The needle is not longer than the haystack.
  __builtin_assume(2 <= sn && sn <= sh);

  // Compute Boyer-Moore's bad-character shift function.
  // Only the last 255 characters of the needle matter for shifts up to 255,
  // which is good enough for most needles.
  size_t n = sn - 1;
  size_t i = 0;
  int c = n;
  if (c >= 255) {
    c = 255;
    i = n - 255;
  }

#ifndef _REENTRANT
  static
#endif
  uint8_t bmbc[256];
  memset(bmbc, c, sizeof(bmbc));
  for (; i < n; i++) {
    // One less than the usual offset because
    // we advance at least one vector at a time.
    size_t t = n - i - 1;
    bmbc[(unsigned char)needle[i]] = t;
  }

  return __memmem_raita(haystk, sh, needle, sn, bmbc);
}

__attribute__((weak))
void *memmem(const void *vh, size_t sh, const void *vn, size_t sn) {
  // Return immediately on empty needle.
  if (sn == 0) return (void *)vh;

  // Return immediately when needle is longer than haystack.
  if (sn > sh) return NULL;

  // Skip to the first matching character using memchr,
  // handling single character needles.
  const char *needle = (char *)vn;
  const char *haystk = (char *)memchr(vh, *needle, sh);
  if (!haystk || sn == 1) return (void *)haystk;

  // The haystack got shorter, is the needle now longer?
  sh -= haystk - (char *)vh;
  if (sn > sh) return NULL;

  return (void *)__memmem(haystk, sh, needle, sn);
}

__attribute__((weak))
char *strstr(const char *haystk, const char *needle) {
  // Return immediately on empty needle.
  if (!needle[0]) return (char *)haystk;

  // Skip to the first matching character using strchr,
  // handling single character needles.
  haystk = strchr(haystk, *needle);
  if (!haystk || !needle[1]) return (char *)haystk;

  return (char *)__memmem(haystk, SIZE_MAX, needle, strlen(needle));
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