#!/bin/bash
# PALDUR (the worker host): one differential, game.exe's FUN_0057d3d0 / FUN_00559e30 against d2d's Fight::wear_*.
#   tools/emu/run-on-worker.sh --ttl 1500 pal_run.sh N SEED > $D2_SCRATCH/../out.txt   (needs pal_build.sh first)
set -e
here=$(cd "$(dirname "$0")" && pwd)
N=${1:-20000}; SEED=${2:-1}
O=${D2_SCRATCH:?run through tools/emu/run-on-worker.sh}
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
UP=$HOME/d2d-lab/src/tools/emu
export PYTHONPATH=$here:$UP
cd $here
$PY pal_wear.py gen $N $SEED > $O/pal_cases.tsv
wc -l $O/pal_cases.tsv
nice $PY pal_wear.py game $O/pal_cases.tsv > $O/pal_game.tsv
$O/pal_wear_test $O/pal_cases.tsv > $O/pal_d2d.tsv
wc -l $O/pal_game.tsv $O/pal_d2d.tsv
md5sum $O/pal_cases.tsv $O/pal_game.tsv $O/pal_d2d.tsv
$PY pal_wear.py compare $O/pal_cases.tsv $O/pal_game.tsv $O/pal_d2d.tsv || true
