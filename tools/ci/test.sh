#!/usr/bin/env bash
# Blocking test run: everything except the quarantined tests (tools/ci/flaky.txt), then each quarantined
# test is run and retried once; it only fails the build if it fails twice.
# Usage: tools/ci/test.sh   (env: PKGS=./..., PAR=4)
set -u
cd "$(dirname "$0")/../.."
pkgs="${PKGS:-./...}"; par="${PAR:-4}"
names=$(grep -v '^\s*#' tools/ci/flaky.txt | awk -F'\t' 'NF>=3{print $1}' | paste -sd'|')
fail=0
if [ -n "$names" ]; then
  go test -short -count=1 -p "$par" -skip "^($names)\$" $pkgs || fail=1
else
  go test -short -count=1 -p "$par" $pkgs || fail=1
fi
if [ -n "$names" ]; then
  # tab-separated: name, package
  while IFS=$'\t' read -r name pkg _; do
    case "$name" in ''|'#'*) continue ;; esac
    [ -n "${pkg:-}" ] || pkg=./...
    go test -short -count=1 -run "^$name\$" "$pkg" && continue
    echo "flaky $name failed, retrying once"
    go test -short -count=1 -run "^$name\$" "$pkg" || { echo "FAIL: flaky test $name failed twice"; fail=1; }
  done < tools/ci/flaky.txt
fi
exit $fail
