#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.5/ddlmod.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.5/ddlmod_test.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.5/error_translator.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.5/migrator.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.5/sqlite.go"
curl -#OL "https://github.com/go-gorm/sqlite/raw/v1.5.5/sqlite_test.go"