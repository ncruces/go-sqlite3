#include <stdint.h>
#include <string.h>
#include <wasm_simd128.h>

int memcmp(const void* vl, const void* vr, size_t n) {
  // Scalar algorithm.
  if (n < sizeof(v128_t)) {
    const unsigned char* u1 = (unsigned char*)vl;
    const unsigned char* u2 = (unsigned char*)vr;
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
  const v128_t* v1 = (v128_t*)vl;
  const v128_t* v2 = (v128_t*)vr;
  while (n) {
    const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(v1), wasm_v128_load(v2));
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(cmp)) {
      // Find the offset of the first zero bit (little-endian).
      size_t ctz = __builtin_ctz(~wasm_i8x16_bitmask(cmp));
      const unsigned char* u1 = (unsigned char*)v1 + ctz;
      const unsigned char* u2 = (unsigned char*)v2 + ctz;
      // This may help the compiler if the function is inlined.
      __builtin_assume(*u1 - *u2 != 0);
      return *u1 - *u2;
    }
    // This makes n a multiple of sizeof(v128_t)
    // for every iteration except the first.
    size_t align = (n - 1) % sizeof(v128_t) + 1;
    v1 = (v128_t*)((char*)v1 + align);
    v2 = (v128_t*)((char*)v2 + align);
    n -= align;
  }
  return 0;
}

void* memchr(const void* s, int c, size_t n) {
  // When n is zero, a function that locates a character finds no occurrence.
  // Otherwise, decrement n to ensure sub_overflow overflows
  // when n would go equal-to-or-below zero.
  if (!n--) {
    return NULL;
  }

  // memchr must behave as if it reads characters sequentially
  // and stops as soon as a match is found.
  // Aligning ensures out of bounds loads are safe.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  uintptr_t addr = (uintptr_t)s - align;
  v128_t vc = wasm_i8x16_splat(c);

  for (;;) {
    v128_t v = *__builtin_launder((v128_t*)addr);
    v128_t cmp = wasm_i8x16_eq(v, vc);
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
        return ctz - align <= n ? (char*)s + (addr - (uintptr_t)s + ctz) : NULL;
      }
    }
    // Decrement n; if it overflows we're done.
    if (__builtin_sub_overflow(n, sizeof(v128_t) - align, &n)) {
      return NULL;
    }
    align = 0;
    addr += sizeof(v128_t);
  }
}

void* memrchr(const void* s, int c, size_t n) {
  // memrchr is allowed to read up to n bytes from the object.
  // Search backward for the last matching character.
  const v128_t* v = (v128_t*)((char*)s + n);
  const v128_t vc = wasm_i8x16_splat(c);
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    const v128_t cmp = wasm_i8x16_eq(wasm_v128_load(--v), vc);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(cmp)) {
      // Find the offset of the last one bit (little-endian).
      // The leading 16 bits of the bitmask are always zero,
      // and to be ignored.
      size_t clz = __builtin_clz(wasm_i8x16_bitmask(cmp)) - 16;
      return (char*)(v + 1) - (clz + 1);
    }
  }

  // Scalar algorithm.
  const char* a = (char*)v;
  while (n--) {
    if (*(--a) == (char)c) return (char*)a;
  }
  return NULL;
}

size_t strlen(const char* s) {
  // strlen must stop as soon as it finds the terminator.
  // Aligning ensures out of bounds loads are safe.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  uintptr_t addr = (uintptr_t)s - align;

  for (;;) {
    v128_t v = *__builtin_launder((v128_t*)addr);
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(v)) {
      const v128_t cmp = wasm_i8x16_eq(v, (v128_t){});
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
        return addr - (uintptr_t)s + __builtin_ctz(mask);
      }
    }
    align = 0;
    addr += sizeof(v128_t);
  }
}

char* strchr(const char* s, int c) {
  char* r = strchrnul(s, c);
  return *r == (char)c ? r : NULL;
}

char* strchrnul(const char* s, int c) {
  // strchrnul must stop as soon as a match is found.
  // Aligning ensures out of bounds loads are safe.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  uintptr_t addr = (uintptr_t)s - align;
  v128_t vc = wasm_i8x16_splat(c);

  for (;;) {
    v128_t v = *__builtin_launder((v128_t*)addr);
    const v128_t cmp = wasm_i8x16_eq(v, (v128_t){}) | wasm_i8x16_eq(v, vc);
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
        return (char*)s + (addr - (uintptr_t)s + __builtin_ctz(mask));
      }
    }
    align = 0;
    addr += sizeof(v128_t);
  }
}

char* strrchr(const char* s, int c) {
  return (char*)memrchr(s, c, strlen(s) + 1);
}

int strcmp(const char* l, const char* r) {
  for (; *l == *r && *l; l++, r++);
  return *(unsigned char*)l - *(unsigned char*)r;
}

int strncmp(const char* _l, const char* _r, size_t n) {
  const unsigned char *l = (void*)_l, *r = (void*)_r;
  if (!n--) return 0;
  for (; *l && *r && n && *l == *r; l++, r++, n--);
  return *l - *r;
}

// SIMDized check which bytes are in a set (Geoff Langdale)
// http://0x80.pl/notesen/2018-10-18-simd-byte-lookup.html

// This is the same algorithm as truffle from Hyperscan:
// https://github.com/intel/hyperscan/blob/v5.4.2/src/nfa/truffle.c#L64-L81
// https://github.com/intel/hyperscan/blob/v5.4.2/src/nfa/trufflecompile.cpp

typedef struct {
  __u8x16 lo;
  __u8x16 hi;
} __wasm_v128_bitmap256_t;

__attribute__((always_inline)) static void __wasm_v128_setbit(
    __wasm_v128_bitmap256_t* bitmap, uint8_t i) {
  uint8_t hi_nibble = i >> 4;
  uint8_t lo_nibble = i & 0xf;
  bitmap->lo[lo_nibble] |= (uint8_t)(1u << (hi_nibble - 0));
  bitmap->hi[lo_nibble] |= (uint8_t)(1u << (hi_nibble - 8));
}

#ifndef __wasm_relaxed_simd__
#define wasm_i8x16_relaxed_swizzle wasm_i8x16_swizzle
#endif

__attribute__((always_inline)) static v128_t __wasm_v128_chkbits(
    __wasm_v128_bitmap256_t bitmap, v128_t v) {
  v128_t hi_nibbles = wasm_u8x16_shr(v, 4);
  v128_t bitmask_lookup = wasm_u64x2_const_splat(0x8040201008040201);
  v128_t bitmask = wasm_i8x16_relaxed_swizzle(bitmask_lookup, hi_nibbles);

  v128_t indices_0_7 = v & wasm_u8x16_const_splat(0x8f);
  v128_t indices_8_15 = indices_0_7 ^ wasm_u8x16_const_splat(0x80);

  v128_t row_0_7 = wasm_i8x16_swizzle((v128_t)bitmap.lo, indices_0_7);
  v128_t row_8_15 = wasm_i8x16_swizzle((v128_t)bitmap.hi, indices_8_15);

  v128_t bitsets = row_0_7 | row_8_15;
  return bitsets & bitmask;
}

#undef wasm_i8x16_relaxed_swizzle

size_t strspn(const char* s, const char* c) {
  // strspn must stop as soon as it finds the terminator.
  // Aligning ensures out of bounds loads are safe.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  uintptr_t addr = (uintptr_t)s - align;

  if (!c[0]) return 0;
  if (!c[1]) {
    v128_t vc = wasm_i8x16_splat(*c);
    for (;;) {
      v128_t v = *__builtin_launder((v128_t*)addr);
      v128_t cmp = wasm_i8x16_eq(v, vc);
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
          return addr - (uintptr_t)s + __builtin_ctz(mask);
        }
      }
      align = 0;
      addr += sizeof(v128_t);
    }
  }

  __wasm_v128_bitmap256_t bitmap = {};

  for (; *c; c++) {
    // Terminator IS NOT on the bitmap.
    __wasm_v128_setbit(&bitmap, (uint8_t)*c);
  }

  for (;;) {
    v128_t v = *__builtin_launder((v128_t*)addr);
    v128_t found = __wasm_v128_chkbits(bitmap, v);
    // Bitmask is slow on AArch64, all_true is much faster.
    if (!wasm_i8x16_all_true(found)) {
      v128_t cmp = wasm_i8x16_eq(found, (v128_t){});
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
        return addr - (uintptr_t)s + __builtin_ctz(mask);
      }
    }
    align = 0;
    addr += sizeof(v128_t);
  }
}

size_t strcspn(const char* s, const char* c) {
  if (!c[0] || !c[1]) return strchrnul(s, *c) - s;

  // strcspn must stop as soon as it finds the terminator.
  // Aligning ensures out of bounds loads are safe.
  uintptr_t align = (uintptr_t)s % sizeof(v128_t);
  uintptr_t addr = (uintptr_t)s - align;

  __wasm_v128_bitmap256_t bitmap = {};

  do {
    // Terminator IS on the bitmap.
    __wasm_v128_setbit(&bitmap, (uint8_t)*c);
  } while (*c++);

  for (;;) {
    v128_t v = *__builtin_launder((v128_t*)addr);
    v128_t found = __wasm_v128_chkbits(bitmap, v);
    // Bitmask is slow on AArch64, any_true is much faster.
    if (wasm_v128_any_true(found)) {
      v128_t cmp = wasm_i8x16_eq(found, (v128_t){});
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
        return addr - (uintptr_t)s + __builtin_ctz(mask);
      }
    }
    align = 0;
    addr += sizeof(v128_t);
  }
}
