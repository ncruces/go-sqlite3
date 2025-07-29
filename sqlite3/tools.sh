#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

ROOT=../

if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
  WASI_SDK="x86_64-windows"
  BINARYEN="x86_64-windows"
elif [[ "$OSTYPE" == "linux"* ]]; then
  if [[ "$(uname -m)" == "x86_64" ]]; then
    WASI_SDK="x86_64-linux"
    BINARYEN="x86_64-linux"
  else
    WASI_SDK="arm64-linux"
    BINARYEN="aarch64-linux"
  fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
  if [[ "$(uname -m)" == "x86_64" ]]; then
    WASI_SDK="x86_64-macos"
    BINARYEN="x86_64-macos"
  else
    WASI_SDK="arm64-macos"
    BINARYEN="arm64-macos"
  fi
fi

WASI_SDK="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-27/wasi-sdk-27.0-$WASI_SDK.tar.gz"
BINARYEN="https://github.com/WebAssembly/binaryen/releases/download/version_123/binaryen-version_123-$BINARYEN.tar.gz"

# Download tools
mkdir -p "$ROOT/tools"
[ -d "$ROOT/tools/wasi-sdk" ] || curl -#L "$WASI_SDK" | tar xzC "$ROOT/tools" &
[ -d "$ROOT/tools/binaryen" ] || curl -#L "$BINARYEN" | tar xzC "$ROOT/tools" &
wait

[ -d "$ROOT/tools/wasi-sdk" ] || mv "$ROOT/tools/wasi-sdk"* "$ROOT/tools/wasi-sdk"
[ -d "$ROOT/tools/binaryen" ] || mv "$ROOT/tools/binaryen"* "$ROOT/tools/binaryen"