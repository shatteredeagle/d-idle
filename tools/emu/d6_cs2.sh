#!/bin/bash
# decomp6 (worker host): covstates2's put differential (cs2_put.py game / d2d-real / model) against the scratch tree of upstream main
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
git -C $T log --oneline -1
cp $HERE/cs2_*.py $HERE/cs2_d2d_test.cpp $T/tools/emu/
export D2_SCRATCH=$O/cs2; mkdir -p $D2_SCRATCH
SRC=$T bash $HERE/cs2_build.sh $B 2>&1 | tail -3
cd $T/tools/emu
N=${1:-100000}; S=${2:-20261102}
P="uv run python cs2_put.py"
$P gen $N $S > $D2_SCRATCH/cases.tsv 2>/dev/null
$P game $D2_SCRATCH/cases.tsv > $D2_SCRATCH/game.tsv 2>/dev/null
nice $D2_SCRATCH/cs2_d2d_test $D2_SCRATCH/cases.tsv > $D2_SCRATCH/d2d.tsv
$P model $D2_SCRATCH/cases.tsv > $D2_SCRATCH/model.tsv 2>/dev/null
wc -l $D2_SCRATCH/*.tsv
$P cmp $D2_SCRATCH/d2d.tsv $D2_SCRATCH/model.tsv d2d-real model 2>&1 | grep -v WARNING | tail -5
$P cmp $D2_SCRATCH/game.tsv $D2_SCRATCH/d2d.tsv game.exe d2d-real 2>&1 | grep -v WARNING | tail -8
