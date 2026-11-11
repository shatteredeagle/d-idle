#!/bin/bash
# seedspawn (the worker host): upstream's diff_drlg.py (game.exe's generator and population in the emulator against d2d's
# drlg-dump) with d2d's side = the binary tools/emu/ss_build.sh built from upstream main.
#   tools/emu/ss_run.sh [LEVELS=2,3,4,8] [THINKS=n] [DIFFICULTY=n] FIRST-LAST LEVEL [kind...]
#   kinds: tiles units seeds monsters objgroups drops collision game npcs layout
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
S=${D2_SCRATCH:-/tmp/d2agent-seedspawn}
BIN=$S/ss-drlg-dump
[ -x "$BIN" ] || { echo "no $BIN: run ss_build.sh first"; exit 1; }
while [[ ${1:-} == LEVELS=* || ${1:-} == THINKS=* || ${1:-} == DIFFICULTY=* ]]; do export "$1"; shift; done
mkdir -p $S/ssbuild/tools/drlg-dump
cp -f "$BIN" $S/ssbuild/tools/drlg-dump/drlg-dump
for t in mpq-cat drop-dump; do   # the game kind reads game data through mpq-cat; drops compare through drop-dump (upstream's builds, read only)
  mkdir -p $S/ssbuild/tools/$t
  [ -x $UP/build/tools/$t/$t ] && cp -f $UP/build/tools/$t/$t $S/ssbuild/tools/$t/$t
done
export D2_BUILD=$S/ssbuild
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
cd "$UP/tools/emu"
exec nice "$PY" diff_drlg.py "$@"
