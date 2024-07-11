#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../../
BINARYEN="$ROOT/tools/binaryen-version_118/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-23.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -flto -g0 -O2 \
	-o speedtest1.wasm main.c \
	-I"$ROOT/sqlite3" \
	-matomics -msimd128 -mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-D_HAVE_SQLITE_CONFIG_H -DSQLITE_USE_URI \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	speedtest1.wasm -o speedtest1.tmp \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext
mv speedtest1.tmp speedtest1.wasm
bzip2 -9f speedtest1.wasm