#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

zig cc --target=wasm32-wasi -flto -g0 -O2 \
  -o speedtest1.wasm main.c \
	-I../../../sqlite3/ \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H