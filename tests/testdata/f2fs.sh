#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"
ROOT=../../

if mountpoint -q f2fs/; then
  sudo umount f2fs/
fi

mkdir -p f2fs/
gunzip -c f2fs.img.gz > f2fs.img
sudo mount -nv -o loop f2fs.img f2fs/
mkdir -p f2fs/tmp/

go test -c "$ROOT/tests" -coverpkg github.com/ncruces/go-sqlite3/...
TMPDIR=f2fs/tmp/ ./tests.test -test.v -test.short -test.coverprofile cover.out
go tool cover -html cover.out

sudo umount f2fs/
rm -r f2fs/ f2fs.img cover.out *.test