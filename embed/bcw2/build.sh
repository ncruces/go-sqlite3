#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../../
BINARYEN="$ROOT/tools/binaryen/bin"
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

trap 'rm -rf build/ sqlite/ bcw2.tmp' EXIT

mkdir -p build/ext/
cp "$ROOT"/sqlite3/*.[ch] build/
cp "$ROOT"/sqlite3/*.patch build/

# https://sqlite.org/src/info/9d6517e7cc8bf175
curl -# https://sqlite.org/src/tarball/sqlite.tar.gz?r=9d6517e7 | tar xz

cd sqlite
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
	MSYS_NO_PATHCONV=1 nmake /f makefile.msc sqlite3.c "OPTS=-DSQLITE_ENABLE_UPDATE_DELETE_LIMIT -DSQLITE_ENABLE_ORDERED_SET_AGGREGATES"
else
	sh configure --enable-update-limit
	OPTS=-DSQLITE_ENABLE_ORDERED_SET_AGGREGATES make sqlite3.c
fi
cd ~-

mv sqlite/sqlite3.c             build/
mv sqlite/sqlite3.h             build/
mv sqlite/sqlite3ext.h          build/
mv sqlite/ext/misc/anycollseq.c build/ext/
mv sqlite/ext/misc/base64.c     build/ext/
mv sqlite/ext/misc/decimal.c    build/ext/
mv sqlite/ext/misc/ieee754.c    build/ext/
mv sqlite/ext/misc/regexp.c     build/ext/
mv sqlite/ext/misc/series.c     build/ext/
mv sqlite/ext/misc/spellfix.c   build/ext/
mv sqlite/ext/misc/uint.c       build/ext/

cd build
cat *.patch | patch -p0 --no-backup-if-mismatch
cd ~-

"$WASI_SDK/clang" --target=wasm32-wasi -std=c23 -g0 -O2 \
	-Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
	-o bcw2.wasm build/main.c \
	-I"$ROOT/sqlite3/libc" -I"build" \
	-mexec-model=reactor \
	-msimd128 -mmutable-globals -mmultivalue \
	-mbulk-memory -mreference-types \
	-mnontrapping-fptoint -msign-ext \
	-fno-stack-protector -fno-stack-clash-protection \
	-Wl,--stack-first \
	-Wl,--import-undefined \
	-Wl,--initial-memory=327680 \
	-D_HAVE_SQLITE_CONFIG_H \
	-DSQLITE_ENABLE_UPDATE_DELETE_LIMIT \
	-DSQLITE_CUSTOM_INCLUDE=sqlite_opt.h \
	$(awk '{print "-Wl,--export="$0}' ../exports.txt)

"$BINARYEN/wasm-ctor-eval" -g -c _initialize bcw2.wasm -o bcw2.tmp
"$BINARYEN/wasm-opt" -g --strip --strip-producers -c -O3 \
	bcw2.tmp -o bcw2.wasm --low-memory-unused \
	--enable-simd --enable-mutable-globals --enable-multivalue \
	--enable-bulk-memory --enable-reference-types \
	--enable-nontrapping-float-to-int --enable-sign-ext