#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../
BINARYEN="$ROOT/tools/binaryen-version_118/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-22.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -flto -g0 -Oz \
	-Wall -Wextra -o sql3parse_table.wasm main.c \
	-mexec-model=reactor \
	-msimd128 -mmutable-globals -mmultivalue \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--export=sql3parse_table

trap 'rm -f sql3parse_table.tmp' EXIT
"$BINARYEN/wasm-ctor-eval" -c _initialize sql3parse_table.wasm -o sql3parse_table.tmp
"$BINARYEN/wasm-opt" --strip --strip-debug --strip-producers -c -Oz \
	sql3parse_table.tmp -o sql3parse_table.wasm \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext