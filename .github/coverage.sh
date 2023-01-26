#!/usr/bin/env bash
set -eo pipefail

SCRIPT_DIR="$(dirname -- "$(readlink -f "${BASH_SOURCE[0]}")")"

go test ./... -coverprofile "$SCRIPT_DIR/coverage.out"
go tool cover -html="$SCRIPT_DIR/coverage.out" -o "$SCRIPT_DIR/coverage.html"
COVERAGE=$(go tool cover -func="$SCRIPT_DIR/coverage.out" | grep total: | grep -Eo '[0-9]+\.[0-9]+')

echo
echo "Full coverage: $COVERAGE% of statements"

COLOR=orange
if (( $(echo "$COVERAGE <= 50" | bc -l) )) ; then
	COLOR=red
elif (( $(echo "$COVERAGE > 80" | bc -l) )); then
	COLOR=green
fi
curl -s "https://img.shields.io/badge/coverage-$COVERAGE%25-$COLOR" > "$SCRIPT_DIR/coverage.svg"

git add "$SCRIPT_DIR/coverage.html" "$SCRIPT_DIR/coverage.svg"