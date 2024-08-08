#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -# https://www.sqlite.org/src/tarball/sqlite.tar.gz?r=bedrock-3.46 | tar xz

cd sqlite
sh configure
make sqlite3.c
cd ~-

mv sqlite/sqlite3.c sqlite/sqlite3.h sqlite/sqlite3ext.h ./
cat *.patch | patch --no-backup-if-mismatch

mkdir -p ext/
mv sqlite/ext/misc/anycollseq.c    ext/
mv sqlite/ext/misc/base64.c        ext/
mv sqlite/ext/misc/decimal.c       ext/
mv sqlite/ext/misc/ieee754.c       ext/
mv sqlite/ext/misc/regexp.c        ext/
mv sqlite/ext/misc/series.c        ext/
mv sqlite/ext/misc/uint.c          ext/

mv sqlite/mptest/mptest.c          ../vfs/tests/mptest/testdata/
mv sqlite/mptest/config01.test     ../vfs/tests/mptest/testdata/
mv sqlite/mptest/config02.test     ../vfs/tests/mptest/testdata/
mv sqlite/mptest/crash01.test      ../vfs/tests/mptest/testdata/
mv sqlite/mptest/crash02.subtest   ../vfs/tests/mptest/testdata/
mv sqlite/mptest/multiwrite01.test ../vfs/tests/mptest/testdata/

mv sqlite/test/speedtest1.c        ../vfs/tests/speedtest1/testdata/

rm -r sqlite