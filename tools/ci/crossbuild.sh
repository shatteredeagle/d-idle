#!/usr/bin/env bash
# Cross-build matrix: go vet + go build of the shipped entry points for every supported GOOS/GOARCH
# (CGO_ENABLED=0). Cheap by design: it checks 6 packages, not the whole repo, and builds to /dev/null.
# Usage: tools/ci/crossbuild.sh [--quick]   (--quick: only windows/amd64 and darwin/arm64)
# Env: CROSSBUILD_PKGS overrides the package list.
set -u
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
pkgs=${CROSSBUILD_PKGS:-"./cmd/d2bot ./cmd/d2dsetup ./cmd/d2dview ./cmd/didleboot ./deploy/d2d-container/d2dping ./apps/desktop"}
targets="linux/amd64 linux/arm64 windows/amd64 darwin/arm64 darwin/amd64"
[ "${1:-}" = --quick ] && targets="windows/amd64 darwin/arm64"
fail=0
for t in $targets; do
  export GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0
  # shellcheck disable=SC2086
  out=$(go vet $pkgs 2>&1 && go build -o /dev/null $pkgs 2>&1) || { echo "FAIL crossbuild $t"; echo "$out" | head -20; fail=1; continue; }
  echo "ok crossbuild $t"
done
exit $fail
