#!/usr/bin/env bash
set -euo pipefail

if [[ "$OSTYPE" == "linux"* ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-21/wasi-sdk-21.0-linux.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_117/binaryen-version_117-x86_64-linux.tar.gz"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-21/wasi-sdk-21.0-macos.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_117/binaryen-version_117-x86_64-macos.tar.gz"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
  WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-21/wasi-sdk-21.0.m-mingw.tar.gz"
  BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_117/binaryen-version_117-x86_64-windows.tar.gz"
fi

# Download tools
mkdir -p tools/
[ -d "tools/wasi-sdk"* ]         || curl -#L "$WASI_SDK" | tar xzC tools &
[ -d "tools/binaryen-version"* ] || curl -#L "$BINARYEN" | tar xzC tools &
wait

sqlite3/download.sh   # Download SQLite
embed/build.sh        # Build Wasm
git diff --exit-code  # Check diffs