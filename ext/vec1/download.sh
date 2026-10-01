#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

DIR=$(mktemp -d)
trap 'rm -rf "$DIR"' EXIT

URL="ftp://ftp.irisa.fr/local/texmex/corpus/siftsmall.tar.gz"
curl -#fL "$URL" | tar xzC "$DIR"

# Run the Go database generator
go run ./build_db.go \
   -out "./siftsmall.db" -limit 2000 \
   -base "$DIR/siftsmall/siftsmall_base.fvecs" \
   -query "$DIR/siftsmall/siftsmall_query.fvecs"

bzip2 -9 siftsmall.db
mv -f siftsmall.db.bz2 testdata/