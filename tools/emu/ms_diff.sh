#!/bin/bash
# monsterstats (the worker host): the whole stat-row differential. Writes game.tsv, d2d.tsv, res.tsv, meta.tsv
# and players.tsv to OUT (default /tmp/ms-run) and prints the comparison counts.
#   tools/emu/ms_diff.sh
set -e
here=$(cd "$(dirname "$0")" && pwd)
OUT=${OUT:-/tmp/ms-run}
mkdir -p $OUT
BIN=$HOME/d2d-lab/pr/ms-out/ms_d2d_test
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
bash $here/ms_build.sh
nice $BIN --stats > $OUT/d2d.tsv
nice $BIN --meta > $OUT/meta.tsv
nice $BIN --res > $OUT/res.tsv
nice $BIN --players > $OUT/players.tsv
wc -l $OUT/d2d.tsv $OUT/meta.tsv $OUT/res.tsv
bash $here/ms_run.sh ms_stats.py game > $OUT/game.tsv
wc -l $OUT/game.tsv
python3 $here/ms_stats.py cmp $OUT/game.tsv $OUT/d2d.tsv $OUT/meta.tsv
md5sum $OUT/game.tsv $OUT/d2d.tsv
