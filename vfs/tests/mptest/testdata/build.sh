#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../../
BINARYEN="$ROOT/tools/binaryen-version_117/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-21.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c17 -flto -g0 -O2 \
  -o mptest.wasm main.c \
	-I"$ROOT/sqlite3" \
	-msimd128 -mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_DEFAULT_SYNCHRONOUS=0 \
	-DSQLITE_DEFAULT_LOCKING_MODE=0 \
	-DHAVE_USLEEP -DSQLITE_NO_SYNC \
	-DSQLITE_THREADSAFE=0 -DSQLITE_OMIT_LOAD_EXTENSION \
	-D_WASI_EMULATED_GETPID -lwasi-emulated-getpid \
  -Wl,--export=aligned_alloc

"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	mptest.wasm -o mptest.tmp \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext
mv mptest.tmp mptest.wasm
bzip2 -9f mptest.wasm