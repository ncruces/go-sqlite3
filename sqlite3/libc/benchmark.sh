#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

echo no SIMD
echo

touch empty.S
./build.sh empty.S &>/dev/null
echo runtime: wazero && go test -bench=.
echo runtime: wasmtime && go test -bench=. -tags=wasmtime
rm -f empty.S

echo with SIMD
echo

./build.sh &>/dev/null
echo runtime: wazero && go test -bench=.
echo runtime: wasmtime && go test -bench=. -tags=wasmtime