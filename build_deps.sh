#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

# download SQLite
url="https://www.sqlite.org/2022/sqlite-amalgamation-3400100.zip"

curl "$url" > sqlite3/sqlite.zip
unzip -d sqlite3/ sqlite3/sqlite.zip
mv sqlite3/sqlite-amalgamation-*/sqlite3* sqlite3/
rm -rf sqlite3/sqlite-amalgamation-*
rm sqlite3/sqlite.zip

# build SQLite
zig cc --target=wasm32-wasi -flto -g0 -O2 -o embed/sqlite3.wasm sqlite3/*.c \
	-DSQLITE_OS_OTHER=1 -DSQLITE_BYTEORDER=1234 \
	-DHAVE_ISNAN -DHAVE_MALLOC_USABLE_SIZE \
	-DSQLITE_DQS=0 \
	-DSQLITE_THREADSAFE=0 \
	-DSQLITE_DEFAULT_MEMSTATUS=0 \
	-DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1 \
	-DSQLITE_LIKE_DOESNT_MATCH_BLOBS \
	-DSQLITE_OMIT_DECLTYPE \
	-DSQLITE_OMIT_DEPRECATED \
	-DSQLITE_OMIT_PROGRESS_CALLBACK \
	-DSQLITE_OMIT_SHARED_CACHE \
	-DSQLITE_OMIT_AUTOINIT \
	-Wl,--export=sqlite3_open_v2 \
	-Wl,--export=sqlite3_close \
	-Wl,--export=sqlite3_prepare_v2 \
	-Wl,--export=sqlite3_exec \
	-Wl,--export=sqlite3_step \
	-Wl,--export=sqlite3_column_text \
	-Wl,--export=sqlite3_column_int64 \
	-Wl,--export=sqlite3_column_double \
	-Wl,--export=sqlite3_errmsg \
	-Wl,--export=malloc \
	-Wl,--export=free
