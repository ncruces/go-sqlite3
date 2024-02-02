#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

go test

rm -rf gorm/ tests/
git clone --branch v1.25.6 --filter=blob:none https://github.com/go-gorm/gorm.git
mv gorm/tests tests
rm -rf gorm/

patch -p1 -N < tests.patch

cd tests
go mod edit \
 -require github.com/ncruces/go-sqlite3/gormlite@v0.0.0 \
 -replace github.com/ncruces/go-sqlite3/gormlite=../ \
 -replace github.com/ncruces/go-sqlite3=../../ \
 -droprequire gorm.io/driver/sqlite \
 -dropreplace gorm.io/gorm
go mod tidy && go work use . && go test

cd ..
rm -rf tests/
go work use -r .