#!/usr/bin/env bash
set -euo pipefail

cd -P -- "$(dirname -- "$0")"

GITHUB_TAG="https://github.com/sqlite/sqlite/raw/version-3.53.4"

cd testdata/
curl -#fOL "$GITHUB_TAG/mptest/config01.test"
curl -#fOL "$GITHUB_TAG/mptest/config02.test"
curl -#fOL "$GITHUB_TAG/mptest/crash01.test"
curl -#fOL "$GITHUB_TAG/mptest/crash02.subtest"
curl -#fOL "$GITHUB_TAG/mptest/multiwrite01.test"
sed -i "s/if vfsname() GLOB 'unix'/if vfsname() GLOB 'os'/" config01.test
cd ~-
