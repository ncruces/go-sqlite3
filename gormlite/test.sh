#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

rm -rf gorm/ tests/
git clone --filter=blob:none --branch=v1.25.1 https://github.com/go-gorm/gorm.git
mv gorm/tests tests
rm -rf gorm/

patch -p1 -N < tests.patch

cd tests
go mod tidy && go test