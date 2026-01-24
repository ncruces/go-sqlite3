#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O2 \
	-o mptest.wasm main.c \
	-I"$ROOT/sqlite3/libc" -I"$ROOT/sqlite3" \
	-mmutable-globals -mnontrapping-fptoint \
	-msimd128 -mbulk-memory -msign-ext \
	-mreference-types -mmultivalue \
	-mno-extended-const \
	-fno-stack-protector \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-D_HAVE_SQLITE_CONFIG_H -DSQLITE_USE_URI \
	-DSQLITE_DEFAULT_SYNCHRONOUS=0 \
	-DSQLITE_DEFAULT_LOCKING_MODE=0 \
	-DSQLITE_NO_SYNC -DSQLITE_THREADSAFE=0 \
	-DSQLITE_OMIT_LOAD_EXTENSION -DHAVE_USLEEP \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	-D_WASI_EMULATED_GETPID -lwasi-emulated-getpid \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-opt" -g mptest.wasm -o mptest.tmp \
	--gufa --generate-global-effects --low-memory-unused --converge -O3 \
	--enable-mutable-globals --enable-nontrapping-float-to-int \
	--enable-simd --enable-bulk-memory --enable-sign-ext \
	--enable-reference-types --enable-multivalue \
	--disable-extended-const \
	--strip --strip-producers
mv mptest.tmp mptest.wasm