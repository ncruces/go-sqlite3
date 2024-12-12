#!/usr/bin/env bash
set -euo pipefail

if [[ "$OSTYPE" == "linux"* ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-25/wasi-sdk-25.0-x86_64-windows.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_120_b/binaryen-version_120_b-x86_64-linux.tar.gz"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-25/wasi-sdk-25.0-arm64-macos.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_120_b/binaryen-version_120_b-arm64-macos.tar.gz"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-25/wasi-sdk-25.0-x86_64-windows.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_120_b/binaryen-version_120_b-x86_64-windows.tar.gz"
fi

# Download tools
mkdir -p tools/
[ -d "tools/wasi-sdk" ] || curl -#L "$WASI_SDK" | tar xzC tools &
[ -d "tools/binaryen" ] || curl -#L "$BINARYEN" | tar xzC tools &
wait

[ -d "tools/wasi-sdk" ] || mv "tools/wasi-sdk"* "tools/wasi-sdk"
[ -d "tools/binaryen" ] || mv "tools/binaryen"* "tools/binaryen"

# Download and build SQLite
sqlite3/download.sh
embed/build.sh
embed/bcw2/build.sh

# Download and build sqlite-createtable-parser
util/sql3util/parse/download.sh
util/sql3util/parse/build.sh

# Check diffs
git diff --exit-code