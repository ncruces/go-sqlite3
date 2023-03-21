#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

# build SQLite
zig cc --target=wasm32-wasi -flto -g0 -Os \
  -o sqlite3.wasm ../sqlite3/main.c \
	-I../sqlite3/ \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H \
	$(awk '{print "-Wl,--export="$0}' exports.txt)

# optimize SQLite
if which wasm-opt; then
	wasm-opt -g -O -o sqlite3.tmp sqlite3.wasm
	mv sqlite3.tmp sqlite3.wasm
fi