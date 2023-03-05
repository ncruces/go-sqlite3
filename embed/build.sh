#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

# download SQLite
../sqlite3/download.sh

# build SQLite
zig cc --target=wasm32-wasi -flto -g0 -Os \
  -o sqlite3.wasm ../sqlite3/amalg.c \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H \
	$(awk '{print "-Wl,--export="$0}' ../sqlite3/exports.txt)