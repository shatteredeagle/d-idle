#!/bin/bash
# monsterstats (the worker host): game.exe's real unit stats for every plain monster of the Act 1 and Act 2
# levels (ms_spawn.py), against d2d's rows. Needs ms_diff.sh run first (d2d.tsv, meta.tsv, res.tsv in OUT).
#   tools/emu/ms_pop.sh SEEDS [DIFFICULTIES]     e.g. tools/emu/ms_pop.sh 1-3 "0 1 2"
set -e
here=$(cd "$(dirname "$0")" && pwd)
OUT=${OUT:-/tmp/ms-run}
SEEDS=${1:-1-3}
DIFFS=${2:-0}
LIDS=${3:-2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
BIN=$HOME/d2d-lab/pr/ms-out/ms_d2d_test
$BIN --bin | grep -E '^[0-9]+ [0-9]+$' > $OUT/bin.tsv
wc -l $OUT/bin.tsv
for d in $DIFFS; do
  bash $here/ms_run.sh ms_spawn.py $SEEDS $LIDS $d | grep -E '^(U|#)' > $OUT/spawn$d.tsv
  grep -c '^U' $OUT/spawn$d.tsv
  grep '^#' $OUT/spawn$d.tsv | head -n 5 || true
  python3 $here/ms_spawn_cmp.py $d $OUT/spawn$d.tsv $OUT/d2d.tsv $OUT/meta.tsv $OUT/res.tsv $OUT/bin.tsv
done
