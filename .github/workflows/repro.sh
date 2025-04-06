#!/usr/bin/env bash
set -euo pipefail

# Download and build SQLite
sqlite3/download.sh
sqlite3/tools.sh
embed/build.sh
embed/bcw2/build.sh

# Download and build sqlite-createtable-parser
util/sql3util/wasm/download.sh
util/sql3util/wasm/build.sh

# Check diffs
git diff --exit-code