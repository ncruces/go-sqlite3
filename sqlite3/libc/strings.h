#ifndef _WASM_SIMD128_STRINGS_H
#define _WASM_SIMD128_STRINGS_H

#include <stddef.h>
#include <wasm_simd128.h>

#include_next <strings.h>  // the system strings.h

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_simd128__

__attribute__((weak))
int bcmp(const void *v1, const void *v2, size_t n) {
  // bcmp is the same as memcmp but only compares for equality.

  const v128_t *w1 = v1;
  const v128_t *w2 = v2;
  for (; n >= sizeof(v128_t); n -= sizeof(v128_t)) {
    // Find any single bit difference.
    if (wasm_v128_any_true(wasm_v128_load(w1) ^ wasm_v128_load(w2))) {
      return 1;
    }
    w1++;
    w2++;
  }

  // Continue byte-by-byte.
  const unsigned char *u1 = (void *)w1;
  const unsigned char *u2 = (void *)w2;
  while (n--) {
    if (*u1 != *u2) return 1;
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