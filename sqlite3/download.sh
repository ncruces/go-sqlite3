#!/usr/bin/env bash
set -eo pipefail

cd -P -- "$(dirname -- "$0")"

if [ ! -f "sqlite3.c" ]; then
	url="https://sqlite.org/2023/sqlite-amalgamation-3410100.zip"
	curl "$url" > sqlite.zip
	unzip -d . sqlite.zip
	mv sqlite-amalgamation-*/sqlite3* .
	rm -rf sqlite-amalgamation-*
	rm sqlite.zip
fi