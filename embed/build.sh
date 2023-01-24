#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

# download SQLite
../sqlite3/download.sh

# build SQLite
zig cc --target=wasm32-wasi -flto -g0 -O2 \
  -o sqlite3.wasm ../sqlite3/*.c \
	-mmutable-globals \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-D_HAVE_SQLITE_CONFIG_H \
	-Wl,--export=malloc \
	-Wl,--export=free \
	-Wl,--export=malloc_destructor \
	-Wl,--export=sqlite3_errcode \
	-Wl,--export=sqlite3_errstr \
	-Wl,--export=sqlite3_errmsg \
	-Wl,--export=sqlite3_error_offset \
	-Wl,--export=sqlite3_open_v2 \
	-Wl,--export=sqlite3_close \
	-Wl,--export=sqlite3_prepare_v3 \
	-Wl,--export=sqlite3_finalize \
	-Wl,--export=sqlite3_reset \
	-Wl,--export=sqlite3_step \
	-Wl,--export=sqlite3_exec \
	-Wl,--export=sqlite3_clear_bindings \
	-Wl,--export=sqlite3_bind_int64 \
	-Wl,--export=sqlite3_bind_double \
	-Wl,--export=sqlite3_bind_text64 \
	-Wl,--export=sqlite3_bind_blob64 \
	-Wl,--export=sqlite3_bind_zeroblob64 \
	-Wl,--export=sqlite3_bind_null \
	-Wl,--export=sqlite3_column_int64 \
	-Wl,--export=sqlite3_column_double \
	-Wl,--export=sqlite3_column_text \
	-Wl,--export=sqlite3_column_blob \
	-Wl,--export=sqlite3_column_bytes \
	-Wl,--export=sqlite3_column_type \
