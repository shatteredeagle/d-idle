#!/bin/bash
# sorc_regress.sh: upstream's own Sorceress missile differential (tools/emu/missile_hit.py, base tree) on base and head d2d (agent palverify).
set -o pipefail
R=$HOME/d2d-lab/palverify; O=$R/out-sorc; mkdir -p $O
export D2_BUILD=$HOME/d2d-lab/src/build D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
cd $R/base/tools/emu && export PYTHONPATH=$PWD
$PY missile_hit.py gen 20000 99 > $O/cases.tsv && nice $PY missile_hit.py game $O/cases.tsv > $O/game.tsv 2> $O/game.err || exit 1
for w in base head; do
  nice ninja -C $R/build-$w -j4 tests/test_missile_hit > /dev/null 2>&1
  $R/build-$w/tests/test_missile_hit --missile-hit $O/cases.tsv | grep -E '^[0-9]+ ' > $O/$w.tsv
  echo "$w: $(wc -l < $O/cases.tsv) cases, $(diff $O/game.tsv $O/$w.tsv | grep -c '^<') differ" >> $O/result.txt
done
