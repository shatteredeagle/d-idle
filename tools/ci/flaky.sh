#!/usr/bin/env bash
# Parses tools/ci/flaky.txt (lines: TestName<TAB>pkg<TAB>owner<TAB>note; '#' comments).
# flaky.sh skip-regex  -> ^(A|B)$ for `go test -skip`
# flaky.sh run-regex   -> same, for `-run`
set -eu
f="$(dirname "$0")/flaky.txt"
names=$(grep -v '^\s*#' "$f" | awk -F'\t' 'NF>=3{print $1}' | paste -sd'|')
echo "^(${names})\$"
