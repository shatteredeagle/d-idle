#!/bin/bash
# SKDEFSHIELDD2D (the worker host): the shield differential. game.exe: sk_melee2.py game (FUN_0057b420 on the stat
# list the equip code builds, skshield-trace.md); d2d: the REAL Fight::update_fighters + player_blow (skdd_test --shield).
# Build first (skdd_build.sh).   tools/emu/run-on-worker.sh --ttl 1500 skdd_shield.sh
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/pr/skmiss}
S=${D2_SCRATCH:?run through tools/emu/run-on-worker.sh}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
python3 "$here/skdd_shield_gen.py" "$S/sc.tsv" "$S/sgc.tsv"
cd "$UP/tools/emu"
nice "$PY" "$here/sk_melee2.py" game "$S/sgc.tsv" > "$S/sgg.tsv"
"$S/skdd_test" --shield "$S/sc.tsv" --keep > "$S/sgr.tsv" 2> "$S/sgr.err"
wc -l "$S/sc.tsv" "$S/sgg.tsv" "$S/sgr.tsv"
python3 -I "$here/skdd_shield_cmp.py" "$S/sc.tsv" "$S/sgg.tsv" "$S/sgr.tsv"
(grep -a "^M " "$S/sgg.tsv" | md5sum; grep -a "^M " "$S/sgr.tsv" | md5sum; md5sum "$S/sc.tsv")
