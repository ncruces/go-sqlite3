#!/usr/bin/env bash
set -euo pipefail

cd tests/testdata
mkdir -p f2fs
gunzip -c f2fs.img.gz > f2fs.img
sudo mount -v -o loop f2fs.img ./f2fs
date > ./f2fs/date
cat ./f2fs/date