#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

go test ../... -coverprofile coverage.svg
COVERAGE=$(go tool cover -func=coverage.svg | grep total: | grep -Eo '[0-9]+\.[0-9]+')
echo $COVERAGE
COLOR=orange
if (( $(echo "$COVERAGE <= 50" | bc -l) )) ; then
	COLOR=red
elif (( $(echo "$COVERAGE > 80" | bc -l) )); then
	COLOR=green
fi
curl -s "https://img.shields.io/badge/coverage-$COVERAGE%25-$COLOR" > coverage.svg
