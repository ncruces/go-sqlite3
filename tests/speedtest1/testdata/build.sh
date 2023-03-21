#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

zig cc --target=wasm32-wasi -flto -g0 -Os \
  -o speedtest1.wasm main.c \
	-I../../../sqlite3/ \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H

if which wasm-opt; then
	wasm-opt -g -O -o speedtest1.tmp speedtest1.wasm
	mv speedtest1.tmp speedtest1.wasm
fi