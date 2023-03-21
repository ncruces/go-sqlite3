#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://sqlite.org/2023/sqlite-amalgamation-3410100.zip"
unzip -d . sqlite-amalgamation-*.zip
mv sqlite-amalgamation-*/sqlite3* .
rm -rf sqlite-amalgamation-*

cd ext/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/ext/misc/decimal.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/ext/misc/uint.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/ext/misc/uuid.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/ext/misc/base64.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/ext/misc/regexp.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/ext/misc/series.c"
cd ~-

cd ../tests/mptest/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/mptest/mptest.c"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/mptest/config01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/mptest/config02.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/mptest/crash01.test"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/mptest/crash02.subtest"
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/mptest/multiwrite01.test"
cd ~-

cd ../tests/speedtest1/testdata/
curl -#OL "https://github.com/sqlite/sqlite/raw/version-3.41.1/test/speedtest1.c"
cd ~-