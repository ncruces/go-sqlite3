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

__attribute__((weak))
int memcmp(const void *v1, const void *v2, size_t n) {
  // Scalar algorithm.
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
        return ctz - align <= n ? (char *)w + ctz : NULL;
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

  // Scalar algorithm.
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
      // If the mask is zero because of alignment,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        return (char *)w - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

static int __strcmp_s(const char *s1, const char *s2) {
  // Scalar algorithm.
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

  return __strcmp_s((char *)w1, (char *)w2);
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

  // Scalar algorithm.
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
  // strchrnul must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator are safe.
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
      // If the mask is zero because of alignment,
      // it's as if we didn't find anything.
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
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (v128_t *)(s - align);

  if (!c[0]) return 0;
  if (!c[1]) {
    const v128_t wc = wasm_i8x16_splat(*c);
    for (;;) {
      const v128_t cmp = wasm_i8x16_eq(*w, wc);
      // Bitmask is slow on AArch64, all_true is much faster.
      if (!wasm_i8x16_all_true(cmp)) {
        // Clear the bits corresponding to alignment (little-endian)
        // so we can count trailing zeros.
        int mask = (uint16_t)~wasm_i8x16_bitmask(cmp) >> align << align;
        // At least one bit will be set, unless we cleared them.
        // Knowing this helps the compiler.
        __builtin_assume(mask || align);
        // If the mask is zero because of alignment,
        // it's as if we didn't find anything.
        if (mask) {
          // Find the offset of the first one bit (little-endian).
          return (char *)w - s + __builtin_ctz(mask);
        }
      }
      align = 0;
      w++;
    }
  }

  __wasm_v128_bitmap256_t bitmap = {};

  for (; *c; c++) {
    // Terminator IS NOT on the bitmap.
    __wasm_v128_setbit(&bitmap, *c);
  }

  for (;;) {
    const v128_t cmp = __wasm_v128_chkbits(bitmap, *w);
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(cmp)) {
      // Clear the bits corresponding to alignment (little-endian)
      // so we can count trailing zeros.
      int mask = (uint16_t)~wasm_i8x16_bitmask(cmp) >> align << align;
      // At least one bit will be set, unless we cleared them.
      // Knowing this helps the compiler.
      __builtin_assume(mask || align);
      // If the mask is zero because of alignment,
      // it's as if we didn't find anything.
      if (mask) {
        // Find the offset of the first one bit (little-endian).
        return (char *)w - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

__attribute__((weak))
size_t strcspn(const char *s, const char *c) {
  if (!c[0] || !c[1]) return __strchrnul(s, *c) - s;

  // strcspn must stop as soon as it finds the terminator.
  // Aligning ensures loads beyond the terminator are safe.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  const v128_t *w = (v128_t *)(s - align);

  __wasm_v128_bitmap256_t bitmap = {};

  do {
    // Terminator IS on the bitmap.
    __wasm_v128_setbit(&bitmap, *c);
  } while (*c++);

  for (;;) {
    const v128_t cmp = __wasm_v128_chkbits(bitmap, *w);
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
        return (char *)w - s + __builtin_ctz(mask);
      }
    }
    align = 0;
    w++;
  }
}

// SIMD-friendly algorithms for substring searching
// http://0x80.pl/notesen/2016-11-28-simd-strfind.html

// For haystacks of known length and large enough needles,
// Boyer-Moore's bad-character rule may be useful,
// as proposed by Horspool, Sunday and Raita.
//
// We augment the SIMD algorithm with Quick Search's
// bad-character shift.
//
// https://igm.univ-mlv.fr/~lecroq/string/node14.html
// https://igm.univ-mlv.fr/~lecroq/string/node18.html
// https://igm.univ-mlv.fr/~lecroq/string/node19.html
// https://igm.univ-mlv.fr/~lecroq/string/node22.html

static const char *__memmem(const char *haystk, size_t sh,  //
                            const char *needle, size_t sn,  //
                            uint8_t bmbc[256]) {
  // We've handled empty and single character needles.
  // The needle is not longer than the haystack.
  __builtin_assume(2 <= sn && sn <= sh);

  // Find the farthest character not equal to the first one.
  size_t i = sn - 1;
  while (i > 0 && needle[0] == needle[i]) i--;
  if (i == 0) i = sn - 1;

  // Subtracting ensures sub_overflow overflows
  // when we reach the end of the haystack.
  if (sh != SIZE_MAX) sh -= sn;

  const v128_t fst = wasm_i8x16_splat(needle[0]);
  const v128_t lst = wasm_i8x16_splat(needle[i]);

  // The last haystack offset for which loading blk_lst is safe.
  const char *H = (char *)(__builtin_wasm_memory_size(0) * PAGESIZE -  //
                           (sizeof(v128_t) + i));

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
        // The match may be after the end of the haystack.
        if (ctz > sh) return NULL;
        // We know the first character matches.
        if (!bcmp(haystk + ctz + 1, needle + 1, sn - 1)) {
          return haystk + ctz;
        }
      }
    }

    size_t skip = sizeof(v128_t);
    if (sh == SIZE_MAX) {
      // Have we reached the end of the haystack?
      if (!wasm_i8x16_all_true(blk_fst)) return NULL;
    } else {
      // Apply the bad-character rule to the character to the right
      // of the righmost character of the search window.
      if (bmbc) skip += bmbc[(unsigned char)haystk[sn - 1 + sizeof(v128_t)]];
      // Have we reached the end of the haystack?
      if (__builtin_sub_overflow(sh, skip, &sh)) return NULL;
    }
    haystk += skip;
  }

  // Scalar algorithm.
  for (size_t j = 0; j <= sh; j++) {
    for (size_t i = 0;; i++) {
      if (sn == i) return haystk;
      if (sh == SIZE_MAX && !haystk[i]) return NULL;
      if (needle[i] != haystk[i]) break;
    }
    haystk++;
  }
  return NULL;
}

__attribute__((weak))
void *memmem(const void *vh, size_t sh, const void *vn, size_t sn) {
  // Return immediately on empty needle.
  if (sn == 0) return (void *)vh;

  // Return immediately when needle is longer than haystack.
  if (sn > sh) return NULL;

  // Skip to the first matching character using memchr,
  // thereby handling single character needles.
  const char *needle = (char *)vn;
  const char *haystk = (char *)memchr(vh, *needle, sh);
  if (!haystk || sn == 1) return (void *)haystk;

  // The haystack got shorter, is the needle now longer than it?
  sh -= haystk - (char *)vh;
  if (sn > sh) return NULL;

  // Is Boyer-Moore's bad-character rule useful?
  if (sn < sizeof(v128_t) || sh - sn < sizeof(v128_t)) {
    return (void *)__memmem(haystk, sh, needle, sn, NULL);
  }

  // Compute Boyer-Moore's bad-character shift function.
  // Only the last 255 characters of the needle matter for shifts up to 255,
  // which is good enough for most needles.
  size_t c = sn;
  size_t i = 0;
  if (c >= 255) {
    i = sn - 255;
    c = 255;
  }

#ifndef _REENTRANT
  static
#endif
  uint8_t bmbc[256];
  memset(bmbc, c, sizeof(bmbc));
  for (; i < sn; i++) {
    // One less than the usual offset because
    // we advance at least one vector at a time.
    bmbc[(unsigned char)needle[i]] = sn - i - 1;
  }

  return (void *)__memmem(haystk, sh, needle, sn, bmbc);
}

__attribute__((weak))
char *strstr(const char *haystk, const char *needle) {
  // Return immediately on empty needle.
  if (!needle[0]) return (char *)haystk;

  // Skip to the first matching character using strchr,
  // thereby handling single character needles.
  haystk = strchr(haystk, *needle);
  if (!haystk || !needle[1]) return (char *)haystk;

  return (char *)__memmem(haystk, SIZE_MAX, needle, strlen(needle), NULL);
}

__attribute__((weak))
char *strcasestr(const char *haystk, const char *needle) {
  // Return immediately on empty needle.
  if (!needle[0]) return (char *)haystk;

  // We've handled empty needles.
  size_t sn = strlen(needle);
  __builtin_assume(sn >= 1);

  // Find the farthest character not equal to the first one.
  size_t i = sn - 1;
  while (i > 0 && needle[0] == needle[i]) i--;
  if (i == 0) i = sn - 1;

  const v128_t fstl = wasm_i8x16_splat(tolower(needle[0]));
  const v128_t fstu = wasm_i8x16_splat(toupper(needle[0]));
  const v128_t lstl = wasm_i8x16_splat(tolower(needle[i]));
  const v128_t lstu = wasm_i8x16_splat(toupper(needle[i]));

  // The last haystk offset for which loading blk_lst is safe.
  const char *H = (char *)(__builtin_wasm_memory_size(0) * PAGESIZE -  //
                           (sizeof(v128_t) + i));

  while (haystk <= H) {
    const v128_t blk_fst = wasm_v128_load((v128_t *)(haystk));
    const v128_t blk_lst = wasm_v128_load((v128_t *)(haystk + i));
    const v128_t eq_fst =
        wasm_i8x16_eq(fstl, blk_fst) | wasm_i8x16_eq(fstu, blk_fst);
    const v128_t eq_lst =
        wasm_i8x16_eq(lstl, blk_lst) | wasm_i8x16_eq(lstu, blk_lst);

    const v128_t cmp = eq_fst & eq_lst;
    if (wasm_v128_any_true(cmp)) {
      // The terminator may come before the match.
      if (!wasm_i8x16_all_true(blk_fst)) break;
      // Find the offset of the first one bit (little-endian).
      // Each iteration clears that bit, tries again.
      for (uint32_t mask = wasm_i8x16_bitmask(cmp); mask; mask &= mask - 1) {
        size_t ctz = __builtin_ctz(mask);
        if (!strncasecmp(haystk + ctz + 1, needle + 1, sn - 1)) {
          return (char *)haystk + ctz;
        }
      }
    }

    // Have we reached the end of the haystack?
    if (!wasm_i8x16_all_true(blk_fst)) return NULL;
    haystk += sizeof(v128_t);
  }

  // Scalar algorithm.
  for (;;) {
    for (size_t i = 0;; i++) {
      if (sn == i) return (char *)haystk;
      if (!haystk[i]) return NULL;
      if (tolower(needle[i]) != tolower(haystk[i])) break;
    }
    haystk++;
  }
  return NULL;
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