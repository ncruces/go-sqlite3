#ifndef _WASM_SIMD128_STRINGS_H
#define _WASM_SIMD128_STRINGS_H

#include <string.h>

#include_next <strings.h>  // the system strings.h

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_simd128__

__attribute__((weak))
int bcmp(const void *v1, const void *v2, size_t n) {
  return __memcmpeq(v1, v2, n);
}

#endif  // __wasm_simd128__

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // _WASM_SIMD128_STRINGS_H