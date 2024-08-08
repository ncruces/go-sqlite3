#!/usr/bin/env bash
set -euo pipefail

if [[ "$OSTYPE" == "linux"* ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-24/wasi-sdk-24.0-x86_64-linux.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_118/binaryen-version_118-x86_64-linux.tar.gz"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-24/wasi-sdk-24.0-x86_64-macos.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_118/binaryen-version_118-x86_64-macos.tar.gz"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-24/wasi-sdk-24.0-x86_64-windows.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_118/binaryen-version_118-x86_64-windows.tar.gz"
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

# Download and build sqlite-createtable-parser
util/vtabutil/parse/download.sh
util/vtabutil/parse/build.sh

# Download and build the bedrock branch (bcw2 patches)
if [[ "$OSTYPE" == "linux"* ]]; then
  embed/bcw2/build.sh
fi

# Check diffs
git diff --exit-code 