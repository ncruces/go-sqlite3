#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -f sqlite3.tmp' EXIT

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O2 \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-matomics -msimd128 -mmutable-globals -mmultivalue \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--initial-memory=327680 \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-ctor-eval" -g -c _initialize sqlite3.wasm -o sqlite3.tmp
"$BINARYEN/wasm-opt" \
	sqlite3.tmp -o sqlite3.wasm \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext \
	--optimize-level 4 \
	--shrink-level 4 \
	--strip-producers \
	--dce --vacuum \
	--flatten \
	--rereloop \
	-Oz -Oz \
	--gufa-optimizing -O4 \
	--inlining-optimizing \
	--precompute-propagate \
	--optimize-instructions \
	--inlining-optimizing \
	--converge
