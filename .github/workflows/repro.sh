#!/usr/bin/env bash
set -euo pipefail

# Download and build SQLite
sqlite3/download.sh
sqlite3/tools.sh
embed/build.sh

# Check diffs
git diff --exit-code