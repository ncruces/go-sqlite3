#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://sqlite.org/2023/sqlite-amalgamation-3420000.zip"
unzip -d . sqlite-amalgamation-*.zip
mv sqlite-amalgamation-*/sqlite3* .
rm -rf sqlite-amalgamation-*

cd ext/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/ext/misc/decimal.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/ext/misc/uint.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/ext/misc/uuid.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/ext/misc/base64.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/ext/misc/regexp.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/ext/misc/series.c"
cd ~-

cd ../sqlite3vfs/tests/mptest/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/mptest/mptest.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/mptest/config01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/mptest/config02.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/mptest/crash01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/mptest/crash02.subtest"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/mptest/multiwrite01.test"
cd ~-

cd ../sqlite3vfs/tests/speedtest1/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.42.0/test/speedtest1.c"
cd ~-