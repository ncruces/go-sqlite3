#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O2 \
	-o speedtest1.wasm main.c \
	-I"$ROOT/sqlite3/libc" -I"$ROOT/sqlite3" \
	-mmutable-globals -mnontrapping-fptoint \
	-msimd128 -mbulk-memory -msign-ext \
	-mreference-types -mmultivalue \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-D_HAVE_SQLITE_CONFIG_H -DSQLITE_USE_URI \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	speedtest1.wasm -o speedtest1.tmp --low-memory-unused \
	--enable-mutable-globals --enable-nontrapping-float-to-int \
	--enable-simd --enable-bulk-memory --enable-sign-ext \
	--enable-reference-types --enable-multivalue
mv speedtest1.tmp speedtest1.wasm