#!/usr/bin/env bash
set -euo pipefail

echo 'set -eu' > test.sh

for p in $(go list ./...); do
  dir=".${p#github.com/ncruces/go-sqlite3}"
  name="$(basename "$p").test"
  (cd ${dir}; go test -c ${BUILDFLAGS:-})
  [ -f "${dir}/${name}" ] && echo "(cd ${dir}; ./${name} ${TESTFLAGS:-})" >> test.sh
done

if [[ -v VMACTIONS ]]; then
  envsubst < .github/actions/vmactions/template.yml > .github/actions/vmactions/action.yml
fi