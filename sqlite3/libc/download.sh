#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

curl -#OL "https://gee.cs.oswego.edu/pub/misc/malloc.c"
