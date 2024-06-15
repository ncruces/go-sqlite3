#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../
BINARYEN="$ROOT/tools/binaryen-version_117/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-22.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c17 -flto -g0 -Oz \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sql3parse_table.wasm sql3parse_table.c \
	-mexec-model=reactor \
	-msimd128 -mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

trap 'rm -f sql3parse_table.tmp' EXIT
"$BINARYEN/wasm-ctor-eval" -g -c _initialize sql3parse_table.wasm -o sql3parse_table.tmp
"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -Oz \
	sql3parse_table.tmp -o sql3parse_table.wasm \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext