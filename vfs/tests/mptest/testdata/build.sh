#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../../
BINARYEN="$ROOT/tools/binaryen-version_113/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-20.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -flto -g0 -O2 \
  -o mptest.wasm main.c \
	-I"$ROOT/sqlite3" \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-DSQLITE_DEFAULT_SYNCHRONOUS=0 \
	-DSQLITE_DEFAULT_LOCKING_MODE=0 \
	-DHAVE_USLEEP -DSQLITE_NO_SYNC \
	-DSQLITE_THREADSAFE=0 -DSQLITE_OMIT_LOAD_EXTENSION \
	-D_WASI_EMULATED_GETPID -lwasi-emulated-getpid

"$BINARYEN/wasm-opt" -g -O2 mptest.wasm -o mptest.tmp \
	--enable-multivalue --enable-mutable-globals \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext
mv mptest.tmp mptest.wasm