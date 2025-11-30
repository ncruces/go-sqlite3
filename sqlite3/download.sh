#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://sqlite.org/2025/sqlite-autoconf-3510100.tar.gz"

# Verify download.
if hash=$(openssl dgst -sha3-256 sqlite-autoconf-*.tar.gz); then
  if ! [[ $hash =~ 9b2b1e73f577def1d5b75c5541555a7f42e6e073ad19f7a9118478389c9bbd9b ]]; then
    echo $hash
    exit 1
  fi
fi 2> /dev/null

tar xzf sqlite-autoconf-*.tar.gz

# To test a snapshot instead:
# curl -# https://sqlite.org/snapshot/sqlite-snapshot-202410081727.tar.gz | tar xz

mv sqlite-*/sqlite3.c .
mv sqlite-*/sqlite3.h .
mv sqlite-*/sqlite3ext.h .
rm -r sqlite-*

GITHUB_TAG="https://github.com/sqlite/sqlite/raw/version-3.51.1"

mkdir -p ext/
cd ext/
curl -#OL "$GITHUB_TAG/ext/misc/anycollseq.c"
curl -#OL "$GITHUB_TAG/ext/misc/base64.c"
curl -#OL "$GITHUB_TAG/ext/misc/decimal.c"
curl -#OL "$GITHUB_TAG/ext/misc/ieee754.c"
curl -#OL "$GITHUB_TAG/ext/misc/regexp.c"
curl -#OL "$GITHUB_TAG/ext/misc/series.c"
curl -#OL "$GITHUB_TAG/ext/misc/spellfix.c"
curl -#OL "$GITHUB_TAG/ext/misc/uint.c"
cd ~-

cd ../vfs/tests/mptest/testdata/
curl -#OL "$GITHUB_TAG/mptest/config01.test"
curl -#OL "$GITHUB_TAG/mptest/config02.test"
curl -#OL "$GITHUB_TAG/mptest/crash01.test"
curl -#OL "$GITHUB_TAG/mptest/crash02.subtest"
curl -#OL "$GITHUB_TAG/mptest/multiwrite01.test"
cd ~-

cd ../vfs/tests/mptest/wasm/
curl -#OL "$GITHUB_TAG/mptest/mptest.c"
cd ~-

cd ../vfs/tests/speedtest1/wasm/
curl -#OL "$GITHUB_TAG/test/speedtest1.c"
cd ~-

cat *.patch | patch -p0 --no-backup-if-mismatch
