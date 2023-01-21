#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

if [ ! -f "sqlite3.c" ]; then
	url="https://www.sqlite.org/2022/sqlite-amalgamation-3400100.zip"
	curl "$url" > sqlite.zip
	unzip -d . sqlite.zip
	mv sqlite-amalgamation-*/sqlite3* .
	rm -rf sqlite-amalgamation-*
	rm sqlite.zip
fi