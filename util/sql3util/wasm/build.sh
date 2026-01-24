#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -f sql3parse_table.tmp' EXIT

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -Oz \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sql3parse_table.wasm main.c \
	-I"$ROOT/sqlite3/libc" -I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-mmutable-globals -mnontrapping-fptoint \
	-msimd128 -mbulk-memory -msign-ext \
	-mreference-types -mmultivalue \
	-mno-extended-const \
	-fno-stack-protector \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--export=sql3parse_table

"$BINARYEN/wasm-ctor-eval" -c _initialize sql3parse_table.wasm -o sql3parse_table.tmp
"$BINARYEN/wasm-opt" sql3parse_table.tmp -o sql3parse_table.wasm \
	--gufa --generate-global-effects --converge -Oz \
	--enable-mutable-globals --enable-nontrapping-float-to-int \
	--enable-simd --enable-bulk-memory --enable-sign-ext \
	--enable-reference-types --enable-multivalue \
	--disable-extended-const \
	--strip --strip-producers
