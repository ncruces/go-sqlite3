#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -f sqlite3.tmp' EXIT

# Run this to generate a new sqlite3.wasm
#
# Then run an test, suggestion:
#
#   go test -v -run ^Example$
#
# I tried various optimization levels.
# Default is: "clang -O2" and "wasm-opt -O3"
# Also tried: "clang -O" and no wasm-opt
#
# With the default ("clang -O2" and "wasm-opt -O3") and the wazero amd64 compiler,
# the first call (sqlite3_malloc64) seems to succeed,
# but the second call (sqlite3_open_v2) panics.
#
# However switching to the interpreter,
# the call to sqlite3_malloc64 returns NULL,
# and I panic with an out of memory.
#
# With "clang -O" and no wasm-opt and the wazero compiler,
# I get a different error on the sqlite3_open_v2 call,
# which does point to an indirect call.
#
# With the interpreter, I get a non-zero bogus return value
# of 4096 (which is definitely not a valid pointer for a malloc result)
# as the first 64K are used by the C stack.

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O3 \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sqlite3.wasm "$ROOT/sqlite3/main.c" \
	-I"$ROOT/sqlite3/libc" -I"$ROOT/sqlite3" \
	-mexec-model=reactor \
	-mmutable-globals -mnontrapping-fptoint \
	-msimd128 -mbulk-memory -msign-ext \
	-mreference-types -mmultivalue -mtail-call \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--initial-memory=327680 \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

"$BINARYEN/wasm-ctor-eval" -g -c _initialize sqlite3.wasm -o sqlite3.tmp

# To disable wasm-opt, run this instead:
# mv sqlite3.tmp sqlite3.wasm

"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	sqlite3.tmp -o sqlite3.wasm --low-memory-unused \
	--enable-mutable-globals --enable-nontrapping-float-to-int \
	--enable-simd --enable-bulk-memory --enable-sign-ext \
	--enable-reference-types --enable-multivalue --enable-tail-call