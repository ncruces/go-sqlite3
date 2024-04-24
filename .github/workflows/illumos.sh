#!/usr/bin/env bash
set -euo pipefail

echo 'set -eu' > test.sh

for p in $(go list ./...); do
  dir=".${p#github.com/ncruces/go-sqlite3}"
  name="$(basename "$p").test"
  (cd ${dir}; GOOS=illumos go test -c)
  [ -f "${dir}/${name}" ] && echo "(cd ${dir}; ./${name} -test.v -test.short)" >> test.sh
done