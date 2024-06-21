#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/ddlmod.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/ddlmod_test.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/error_translator.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/migrator.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/sqlite.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/sqlite_test.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.6/sqlite_test.go"
curl -#L "https://github.com/glebarez/sqlite/raw/v1.11.0/sqlite_error_translator_test.go" > error_translator_test.go