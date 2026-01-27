#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://github.com/ncruces/sqlite-createtable-parser/raw/master/sql3parse_table.c"
curl -#OL "https://github.com/ncruces/sqlite-createtable-parser/raw/master/sql3parse_table.h"
