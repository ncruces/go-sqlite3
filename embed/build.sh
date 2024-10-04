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

# For more information on arguments passed to
# wasm-opt please see `wasm-opt --help` and:
# https://github.com/WebAssembly/binaryen/wiki/Optimizer-Cookbook
#
# --debuginfo            : emit "names" section (useful during Go panics)
# --optimize-level 4     : set code optimization level to max
# --shrink-level 4       : set code shrinking level to max
# --strip-producers      : strip the wasm "producers" section
# --dce                  : dead code elimination
# --vacuum               : remove more obviously un-needed code
# --type-ssa             : creates new nominal types to help optimizations
# -Oz                    : a combined set of optimization passes focused on *size*
# --type-merging         : merges to get rid of above used ssa types
# -O3                    : a combined set of optimization passes focused on *speed*
# --converge             : re-runs the whole set of passes while binary size keeps shrinking
"$BINARYEN/wasm-opt" --debuginfo \
	sqlite3.tmp -o sqlite3.wasm \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext \
	--optimize-level 4 \
	--shrink-level 4 \
	--strip-producers \
	--dce --vacuum \
	--type-ssa \
	-Oz \
	--type-merging \
	-O3 \
	--converge
