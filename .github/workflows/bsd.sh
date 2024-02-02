#!/usr/bin/env bash

echo 'set -euo pipefail' > test.sh

for p in $(go list ./...); do
  dir=".${p#github.com/ncruces/go-sqlite3}"
  name="$(basename "$p").test"
  (cd ${dir}; GOOS=freebsd go test -c)
  [ -f "${dir}/${name}" ] && echo "(cd ${dir}; ./${name} -test.v)" >> test.sh
done