#!/usr/bin/env bash
set -euo pipefail

# Download and build SQLite
sqlite3/download.sh
sqlite3/tools.sh
embed/build.sh
embed/bcw2/build.sh

# Check diffs
git diff --exit-code