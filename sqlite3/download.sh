#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://sqlite.org/2025/sqlite-amalgamation-3480000.zip"
unzip -d . sqlite-amalgamation-*.zip
mv sqlite-amalgamation-*/sqlite3.c .
mv sqlite-amalgamation-*/sqlite3.h .
mv sqlite-amalgamation-*/sqlite3ext.h .
rm -rf sqlite-amalgamation-*

# To test a snapshot:
# curl -# https://sqlite.org/snapshot/sqlite-snapshot-202410081727.tar.gz | tar xz
# mv sqlite-snapshot-*/sqlite3.c .
# mv sqlite-snapshot-*/sqlite3.h .
# mv sqlite-snapshot-*/sqlite3ext.h .
# rm -rf sqlite-snapshot-*

mkdir -p ext/
cd ext/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/anycollseq.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/base64.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/decimal.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/ieee754.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/regexp.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/series.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/spellfix.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/ext/misc/uint.c"
cd ~-

cd ../vfs/tests/mptest/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/mptest/config01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/mptest/config02.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/mptest/crash01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/mptest/crash02.subtest"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/mptest/multiwrite01.test"
cd ~-

cd ../vfs/tests/mptest/wasm/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/mptest/mptest.c"
cd ~-

cd ../vfs/tests/speedtest1/wasm/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.48.0/test/speedtest1.c"
cd ~-

cat *.patch | patch -p0 --no-backup-if-mismatch