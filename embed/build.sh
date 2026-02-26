#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -f sqlite3.tmp' EXIT

"$WASI_SDK/clang" --target=wasm32 -nostdlib -std=c23 -g0 -O2 \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3/libc" -I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-mmutable-globals -mbulk-memory -mmultivalue \
	-msign-ext -mnontrapping-fptoint \
	-mno-simd128 -mno-extended-const \
	-fno-stack-protector \
	-Wl,--stack-first \
	-Wl,--import-memory \
	-Wl,--import-undefined \
	-Wl,--initial-memory=327680 \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_EXPERIMENTAL_PRAGMA_20251114 \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

mv sqlite3.wasm sqlite3.tmp

"$BINARYEN/wasm-opt" -g sqlite3.tmp -o sqlite3.wasm \
	--gufa --generate-global-effects --low-memory-unused --converge -O3 \
	--enable-mutable-globals --enable-bulk-memory --enable-multivalue \
	--enable-sign-ext --enable-nontrapping-float-to-int \
	--disable-simd --disable-extended-const \
	--strip --strip-producers
