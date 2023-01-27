#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(dirname -- "$(readlink -f "${BASH_SOURCE[0]}")")"

go test ./... -coverprofile "$SCRIPT_DIR/coverage.out"
go tool cover -html="$SCRIPT_DIR/coverage.out" -o "$SCRIPT_DIR/coverage.html"
COVERAGE=$(go tool cover -func="$SCRIPT_DIR/coverage.out" | tail -1 | grep -Eo '\d+\.\d')

echo "coverage: $COVERAGE% of statements"

COLOR=orange
if awk "BEGIN {exit !($COVERAGE <= 50)}"; then
	COLOR=red
elif awk "BEGIN {exit !($COVERAGE > 80)}"; then
	COLOR=green
fi
curl -s "https://img.shields.io/badge/coverage-$COVERAGE%25-$COLOR" > "$SCRIPT_DIR/coverage.svg"

git add "$SCRIPT_DIR/coverage.html" "$SCRIPT_DIR/coverage.svg"