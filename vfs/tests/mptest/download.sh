#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

GITHUB_TAG="https://github.com/sqlite/sqlite/raw/version-3.52.0"

cd testdata/
curl -#OL "$GITHUB_TAG/mptest/config01.test"
curl -#OL "$GITHUB_TAG/mptest/config02.test"
curl -#OL "$GITHUB_TAG/mptest/crash01.test"
curl -#OL "$GITHUB_TAG/mptest/crash02.subtest"
curl -#OL "$GITHUB_TAG/mptest/multiwrite01.test"
cd ~-
