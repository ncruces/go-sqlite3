#!/usr/bin/env bash
cd -P -- "$(dirname -- "$0")"

shopt -s extglob
clang-format -i !(sqlite3*).@(c|h)