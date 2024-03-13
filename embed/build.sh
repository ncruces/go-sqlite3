#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen-version_117/bin"
WASI_SDK="$ROOT/tools/wasi-sdk-21.0/bin"

"$WASI_SDK/clang" --target=wasm32-wasi-threads -pthread -std=c17 -flto -g0 -O2 \
  -Wall -Wextra -Wno-unused-parameter \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-matomics -msimd128 -mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--import-undefined \
	-Wl,--global-base=1024 \
	-Wl,--import-memory -Wl,--export-memory -Wl,--max-memory=4294967296 \
	-D_HAVE_SQLITE_CONFIG_H \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	sqlite3.wasm -o sqlite3.tmp \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext
mv sqlite3.tmp sqlite3.wasm