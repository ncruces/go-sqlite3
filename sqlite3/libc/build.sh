#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -f libc.tmp' EXIT

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O2 \
	-o libc.wasm ../strings.c \
	-mexec-model=reactor \
	-msimd128 -mmutable-globals -mmultivalue \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--initial-memory=16777216 \
	-Wl,--export=memset \
	-Wl,--export=memcpy \
	-Wl,--export=memcmp

"$BINARYEN/wasm-ctor-eval" -g -c _initialize libc.wasm -o libc.tmp
"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	libc.tmp -o libc.wasm \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext

"$BINARYEN/wasm-dis" -o libc.wat libc.wasm