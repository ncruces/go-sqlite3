#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://sqlite.org/2024/sqlite-amalgamation-3460100.zip"
unzip -d . sqlite-amalgamation-*.zip
mv sqlite-amalgamation-*/sqlite3* .
rm -rf sqlite-amalgamation-*

cat *.patch | patch --no-backup-if-mismatch

mkdir -p ext/
cd ext/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/anycollseq.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/base64.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/decimal.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/ieee754.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/regexp.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/series.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/spellfix.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/ext/misc/uint.c"
cd ~-

cd ../vfs/tests/mptest/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/mptest/mptest.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/mptest/config01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/mptest/config02.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/mptest/crash01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/mptest/crash02.subtest"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/mptest/multiwrite01.test"
cd ~-

cd ../vfs/tests/speedtest1/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.46.1/test/speedtest1.c"
cd ~-