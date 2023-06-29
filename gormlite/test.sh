#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

rm -rf gorm/ tests/ $TMPDIR/gorm.db
git clone --filter=blob:none https://github.com/go-gorm/gorm.git
mv gorm/tests tests
rm -rf gorm/

patch -p1 -N < tests.patch

cd tests
go mod tidy && go work use . && go test

cd ..
rm -rf tests/ $TMPDIR/gorm.db
go work use -r .