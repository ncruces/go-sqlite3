#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

zig cc --target=wasm32-wasi -flto -g0 -O2 \
  -o mptest.wasm main.c \
	-I../../../../../sqlite3/ \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_DEFAULT_SYNCHRONOUS=0 \
	-DSQLITE_DEFAULT_LOCKING_MODE=0 \
	-DHAVE_USLEEP -DSQLITE_NO_SYNC \
	-DSQLITE_THREADSAFE=0 -DSQLITE_OMIT_LOAD_EXTENSION \
	-D_WASI_EMULATED_GETPID -lwasi-emulated-getpid