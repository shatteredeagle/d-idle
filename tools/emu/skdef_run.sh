#!/bin/bash
# skdefenders (the worker host): the whole game.exe side of the defender differential.
#   tools/emu/skdef_run.sh N SEED   -> $D2_SCRATCH/c$SEED.tsv (cases), g$SEED.tsv (game.exe), p$SEED.tsv (d2d port)
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/pr/skmiss}
S=${D2_SCRATCH:-/tmp/d2agent-skdefenders}
mkdir -p "$S"
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
N=${1:-1000}
SEED=${2:-1}
cd "$UP/tools/emu"
nice "$PY" "$here/skdef_resist.py" gen "$N" "$SEED" > "$S/c$SEED.tsv"
wc -l "$S/c$SEED.tsv"
nice "$PY" "$here/skdef_resist.py" game "$S/c$SEED.tsv" > "$S/g$SEED.tsv"
nice "$PY" "$here/skdef_resist.py" port "$S/c$SEED.tsv" > "$S/p$SEED.tsv"
wc -l "$S/g$SEED.tsv" "$S/p$SEED.tsv"
head -5 "$S/g$SEED.tsv"
