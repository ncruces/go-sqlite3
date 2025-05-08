#ifndef _WASM_SIMD128_STRINGS_H
#define _WASM_SIMD128_STRINGS_H

#include <stddef.h>
#include <wasm_simd128.h>

#include_next <strings.h>  // the system strings.h

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_simd128__
#ifndef __OPTIMIZE_SIZE__

__attribute__((weak))
int bcmp(const void *v1, const void *v2, size_t n) {
  // bcmp is the same as memcmp but only compares for equality.

  // Baseline algorithm.
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
#endif  // __wasm_simd128__

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // _WASM_SIMD128_STRINGS_H