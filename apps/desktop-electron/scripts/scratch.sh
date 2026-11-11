#!/bin/sh
# Run a command in a scratch copy of this folder so node_modules survives labrun's re-sync (which deletes untracked
# files). Usage on the worker host, from the synced worktree: sh apps/desktop-electron/scripts/scratch.sh npm test
set -eu
src=$(cd "$(dirname "$0")/.." && pwd)
dst=${D2E_SCRATCH:-$HOME/scratch/d2idle-electron}
mkdir -p "$dst"
rsync -a --delete --exclude node_modules --exclude dist "$src"/ "$dst"/
cd "$dst"
exec "$@"
