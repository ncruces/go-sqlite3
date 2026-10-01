#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

rm -rf tests/
go work use -r .
go test

curl -#fL https://github.com/go-gorm/gorm/archive/refs/tags/v1.31.2.tar.gz |\
  tar -vxz --strip-components=1 gorm-1.31.2/tests/

patch -p1 -N < tests.patch
exit

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