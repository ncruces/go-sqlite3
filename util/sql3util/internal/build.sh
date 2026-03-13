#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

curl -#OL "https://github.com/ncruces/sqlite-createtable-parser/raw/master/sql3parse_table.c"
curl -#OL "https://github.com/ncruces/sqlite-createtable-parser/raw/master/sql3parse_table.h"

trap 'rm -f sql3parse_table.*' EXIT

"$WASI_SDK/clang" --target=wasm32 -nostdlib -std=c23 -g0 -Oz \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o sql3parse_table main.c \
	-I"$ROOT/sqlite3/libc" \
	-mexec-model=reactor \
	-mmutable-globals -mmultivalue \
	-mbulk-memory -mreference-types \
	-msign-ext -mnontrapping-fptoint \
	-mno-simd128 -mno-extended-const \
	-fno-stack-protector \
	-Wl,--no-entry \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--export=sql3parse_table

mv sql3parse_table sql3parse_table.tmp

"$BINARYEN/wasm-opt" -g sql3parse_table.tmp -o sql3parse_table.wasm \
	--gufa --generate-global-effects --converge -Oz \
	--enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-sign-ext --enable-nontrapping-float-to-int \
	--disable-simd --disable-extended-const \
	--strip --strip-producers

go run github.com/ncruces/wasm2go@v0.1.0 < sql3parse_table.wasm > parser/parser.go
