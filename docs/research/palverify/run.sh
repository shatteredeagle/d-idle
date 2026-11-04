#!/bin/bash
# run.sh SEED N: generate cases, run game.exe's oracle and both d2d probes, compare (agent palverify, worker host).
set -o pipefail
R=$HOME/d2d-lab/palverify; S=$1; N=$2; O=$R/out-$S; mkdir -p $O
export D2_BUILD=$HOME/d2d-lab/src/build D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
cd $R/base/tools/emu && export PYTHONPATH=$PWD
$PY $R/h/hammer_game.py gen $S $N > $O/cases.tsv || exit 1
nice $PY $R/h/hammer_game.py game $O/cases.tsv > $O/game.tsv 2> $O/game.err || exit 1
PV_DIRECT=1 nice $PY $R/h/hammer_game.py game $O/cases.tsv > $O/gamedirect.tsv 2> $O/gamedirect.err || exit 1
for w in base head; do
  $R/build-$w/tests/test_palprobe --hammer $O/cases.tsv 2> $O/$w.err | grep -E '^[0-9]+ ' > $O/$w.tsv || exit 1
  PV_PREDRAW=1 $R/build-$w/tests/test_palprobe --hammer $O/cases.tsv 2> /dev/null | grep -E '^[0-9]+ ' > $O/$w-predraw.tsv || exit 1
done
echo "== direct: game.exe without FUN_005a89a0 vs d2d one draw" > $O/cmp.txt
$PY $R/h/cmp.py $O/cases.tsv $O/gamedirect.tsv $O/base.tsv $O/head.tsv >> $O/cmp.txt
echo "== full hit path: game.exe FUN_005a89a0 + dmg func vs d2d with the first draw advanced" >> $O/cmp.txt
$PY $R/h/cmp.py $O/cases.tsv $O/game.tsv $O/base-predraw.tsv $O/head-predraw.tsv >> $O/cmp.txt
echo "== full hit path vs d2d one draw (as d2d runs)" >> $O/cmp.txt
$PY $R/h/cmp.py $O/cases.tsv $O/game.tsv $O/base.tsv $O/head.tsv >> $O/cmp.txt
