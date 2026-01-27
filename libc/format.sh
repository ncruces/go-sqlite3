#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../
WASI_SDK="$ROOT/tools/wasi-sdk/bin"

"$WASI_SDK/clang-format" --style=Google -i *.c *.h
