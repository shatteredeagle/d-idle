#!/usr/bin/env bash
# Offline test of deploy-window.sh in a scratch repo (no sandbox, no labrun).
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
t=$(mktemp -d); [ -n "${KEEP:-}" ] || trap 'rm -rf "$t"' EXIT
export D2_DEPLOY_STATE=$t/state D2_DEPLOY_REF=main
git init -q -b main "$t/r"; cd "$t/r"
g() { git -c user.name=t -c user.email=t@t "$@"; }
g commit -q --allow-empty -m base
mkdir -p tools; printf '#!/bin/sh\necho DEPLOYED "$@"\n' >tools/dev-update.sh; chmod +x tools/dev-update.sh
cp "$here/deploy-window.sh" "$t/dw.sh"; mkdir -p "$t/tools/play-sandbox"; cp "$t/dw.sh" "$t/tools/play-sandbox/deploy-window.sh"
ln -s "$t/r/tools/dev-update.sh" "$t/tools/dev-update.sh"; ln -s "$t/r/.git" "$t/.git"
dw=$t/tools/play-sandbox/deploy-window.sh
$dw check && { echo "FAIL: due with no merges"; exit 1; }
g commit -q --allow-empty -m "Merge agent/a: x"
$dw check && { echo "FAIL: due with 1 merge"; exit 1; }
g commit -q --allow-empty -m "Merge agent/b: y"
$dw check >/dev/null || { echo "FAIL: not due with 2"; exit 1; }
out=$($dw run --to c1); case $out in *DEPLOYED*) ;; *) echo "FAIL: run"; exit 1 ;; esac
$dw check && { echo "FAIL: due after deploy"; exit 1; }
$dw request now please >/dev/null; $dw check >/dev/null || { echo "FAIL: request"; exit 1; }
out=$($dw run --to c1 --dry-run); case $out in *"dry run"*) ;; *) echo "FAIL: dry"; exit 1 ;; esac
echo "deploy-window test ok"
