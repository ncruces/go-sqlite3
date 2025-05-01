#ifndef _WASM_SIMD128_MATH_H
#define _WASM_SIMD128_MATH_H

#include <wasm_simd128.h>

#include_next <math.h>  // the system math.h

#ifdef __cplusplus
extern "C" {
#endif

#ifdef __wasm_relaxed_simd__

// This header assumes "relaxed fused multiply-add"
// is both faster and more precise.

#define FP_FAST_FMA 1

__attribute__((weak))
double fma(double x, double y, double z) {
  // If we get a software implementation from the host,
  // this is enough to short circuit it on the 2nd lane.
  const v128_t wx = wasm_f64x2_replace_lane(b, 0, x);
  const v128_t wy = wasm_f64x2_splat(y);
  const v128_t wz = wasm_f64x2_splat(z);
	const v128_t wr = wasm_f64x2_relaxed_madd(wx, wy, wz);
	return wasm_f64x2_extract_lane(wr, 0);
}

#endif  // __wasm_relaxed_simd__

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // _WASM_SIMD128_MATH_H