#!/bin/bash
# SKDEFSHIELDD2D (the worker host): Amplify Damage / Decrepify on the player and the hireling, game.exe
# (skdef_resist.py game, the curse in full on stat 0x24) against d2d's REAL curse path (skdd_test --curse).
#   tools/emu/run-on-worker.sh --ttl 1500 skdd_curse.sh
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/pr/skmiss}
S=${D2_SCRATCH:?run through tools/emu/run-on-worker.sh}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
python3 -I "$here/skdd_curse_gen.py" > "$S/cc.tsv"
wc -l "$S/cc.tsv"
cd "$UP/tools/emu"
nice "$PY" "$here/skdef_resist.py" game "$S/cc.tsv" > "$S/cg.tsv"
"$S/skdd_test" --defend "$S/cc.tsv" --curse > "$S/cr.tsv" 2> "$S/cr.err"
wc -l "$S/cg.tsv" "$S/cr.tsv"
python3 -I "$here/skdd_cmp.py" "$S/cc.tsv" "$S/cg.tsv" "$S/cr.tsv" list
md5sum "$S/cc.tsv" "$S/cg.tsv"
grep -a -E "^[0-9]+ " "$S/cr.tsv" | md5sum
