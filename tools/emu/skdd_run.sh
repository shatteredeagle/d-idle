#!/bin/bash
# SKDEFSHIELDD2D (the worker host): the game.exe side (skdef_run.sh: cases c$SEED.tsv, game g$SEED.tsv),
# then the REAL-d2d probe (skdd_build.sh first) and the comparison.
#   tools/emu/run-on-worker.sh --ttl 3000 skdd_run.sh N SEED [skipgame]
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:?run through tools/emu/run-on-worker.sh}
N=${1:-2000}
SEED=${2:-1}
if [ "$3" != "skipgame" ]; then bash "$here/skdef_run.sh" "$N" "$SEED"; fi
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
"$S/skdd_test" --defend "$S/c$SEED.tsv" --keep > "$S/r$SEED.tsv" 2> "$S/r$SEED.err"
wc -l "$S/r$SEED.tsv"
head -3 "$S/r$SEED.err"
echo "== all compared cases (player: aura / curse values through Fight::Boost; hireling: none)"
"$PY" -I "$here/skdd_cmp.py" "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/r$SEED.tsv"
echo "== core: no aura / curse value"
"$PY" -I "$here/skdd_cmp.py" "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/r$SEED.tsv" core
echo "== player aura / curse split"
"$PY" -I "$here/skdd_ana.py" "$S/c$SEED.tsv" "$S/g$SEED.tsv" "$S/r$SEED.tsv" | grep -v "^  ex"
md5sum "$S/c$SEED.tsv" "$S/g$SEED.tsv"
grep -a -E "^[0-9]+ " "$S/r$SEED.tsv" | md5sum
