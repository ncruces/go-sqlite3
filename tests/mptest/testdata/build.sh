#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

if [ ! -f "mptest.c" ]; then
	curl -sOL	"https://github.com/sqlite/sqlite/raw/master/mptest/mptest.c"
	curl -sOL "https://github.com/sqlite/sqlite/raw/master/mptest/config01.test"
  curl -sOL "https://github.com/sqlite/sqlite/raw/master/mptest/config02.test"
  curl -sOL "https://github.com/sqlite/sqlite/raw/master/mptest/crash01.test"
  curl -sOL "https://github.com/sqlite/sqlite/raw/master/mptest/crash02.subtest"
  curl -sOL "https://github.com/sqlite/sqlite/raw/master/mptest/multiwrite01.test"
fi

zig cc --target=wasm32-wasi -flto -g0 -Os \
  -o mptest.wasm main.c test.c \
	-I../../../sqlite3 \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_DEFAULT_LOCKING_MODE=0 \
	-DHAVE_USLEEP -DSQLITE_NO_SYNC \
	-DSQLITE_THREADSAFE=0 -DSQLITE_OMIT_LOAD_EXTENSION \
	-D_WASI_EMULATED_GETPID -lwasi-emulated-getpid