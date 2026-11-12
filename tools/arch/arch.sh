#!/usr/bin/env bash
# Regenerate the d.idle architecture diagrams with Archify (docs/arch.md).
#   tools/arch/arch.sh                       overview + every-package diagram of this checkout
#   tools/arch/arch.sh --diff OLD NEW        drift between two git refs (e.g. v0.9.0 main)
#   tools/arch/arch.sh --repo DIR [--name N] diagrams of another Go checkout (fork, ops repo)
# Output: docs/arch/out/ (gitignored), one self-contained HTML per diagram.
# Archify is a tool, never a dependency: it is fetched by commit into ~/.cache/archify and
# run with its update check off; nothing here talks to the network except that one fetch.
set -euo pipefail
ARCHIFY_REPO=https://github.com/tt-a1i/archify
ARCHIFY_SHA=7e45f36c66de1254d5ad6ba1d878f6499ea269fd   # pinned, 2026-11-10 stable (v3.2.0, Apache-2.0)
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd "$HERE/../.." && pwd)
OUT=${ARCH_OUT:-$ROOT/docs/arch/out}
CACHE=${ARCHIFY_HOME:-$HOME/.cache/archify}/$ARCHIFY_SHA
export ARCHIFY_UPDATE_CHECK_DISABLED=1

install_archify() {
  [ -f "$CACHE/archify/bin/archify.mjs" ] && return
  command -v node >/dev/null || { echo "arch: node 18+ is required" >&2; exit 2; }
  mkdir -p "$CACHE"
  ( cd "$CACHE" && git init -q . && git remote add origin "$ARCHIFY_REPO" 2>/dev/null || true
    git config core.sparseCheckout true && echo '/archify/' > .git/info/sparse-checkout
    git fetch -q --depth 1 origin "$ARCHIFY_SHA" && git checkout -q FETCH_HEAD )
  [ "$(git -C "$CACHE" rev-parse HEAD)" = "$ARCHIFY_SHA" ] || { echo "arch: pin mismatch" >&2; exit 2; }
}

render() { # <repo-dir> <mode> <name>
  local j="$OUT/$3.json"
  nice python3 -I "$HERE/gen.py" "$1" "$2" "$j" "$3.html" >/dev/null
  ( cd "$OUT" && nice node "$CACHE/archify/bin/archify.mjs" render architecture "$3.json" "$3.html" >/dev/null )
  rm -f "$j"
  echo "$OUT/$3.html"
}

mode=gen; repo=$ROOT; name=dnn; old=; new=
while [ $# -gt 0 ]; do
  case $1 in
    --diff) mode=diff; old=$2; new=$3; shift 3 ;;
    --repo) repo=$(cd "$2" && pwd); shift 2 ;;
    --name) name=$2; shift 2 ;;
    -h|--help) sed -n '2,8p' "$0"; exit 0 ;;
    *) echo "arch: unknown argument $1" >&2; exit 2 ;;
  esac
done
[ "$name" = dnn ] && { [ "$repo" = "$ROOT" ] && name=d2lod-bot || name=$(basename "$repo"); }
install_archify
mkdir -p "$OUT"
if [ $mode = gen ]; then
  render "$repo" overview "$name-overview"
  render "$repo" packages "$name-packages"
else
  tmp=$(mktemp -d); trap 'git -C "$ROOT" worktree remove --force "$tmp/o" "$tmp/n" 2>/dev/null; rm -rf "$tmp"' EXIT
  git -C "$ROOT" worktree add -q --detach "$tmp/o" "$old"
  git -C "$ROOT" worktree add -q --detach "$tmp/n" "$new"
  nice python3 -I "$HERE/gen.py" "$tmp/o" overview "$tmp/base.json" drift.html >/dev/null
  nice python3 -I "$HERE/gen.py" "$tmp/n" overview "$tmp/head.json" drift.html >/dev/null
  safe=$(printf '%s-%s' "$old" "$new" | tr -c 'A-Za-z0-9.\n_-' '_')
  ( cd "$tmp" && nice node "$CACHE/archify/bin/archify.mjs" compare architecture base.json head.json "$OUT/drift-$safe.html" >/dev/null )
  echo "$OUT/drift-$safe.html"
fi
