#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

shopt -s extglob
clang-format -i !(sqlite3*).@(c|h)