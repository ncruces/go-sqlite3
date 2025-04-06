#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

touch empty.S
./build.sh empty.S
go test -bench=.
rm -f empty.S

./build.sh
go test -bench=.