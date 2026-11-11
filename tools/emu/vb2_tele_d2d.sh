#!/bin/bash
# verifyb2 (the worker host): the d2d side of the warp spot search on a game.exe case set.
#   vb2_tele_d2d.sh SEED TREE
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifyb2}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
SEED=$1; t=$2
[ -x "$HOME/d2d-lab/vb2/bin/vb2_tele_probe-$t" ] || bash "$here/vb2_probe_build.sh" "$t" vb2_tele_probe
nice -n 19 "$HOME/d2d-lab/vb2/bin/vb2_tele_probe-$t" "$S/tele-$SEED.cases.tsv" > "$S/tele-$SEED.d2d-$t.tsv"
echo "== $t vs game.exe"
bash "$here/run-py.sh" vb2_tele.py cmp "$S/tele-$SEED.cases.tsv" "$S/tele-$SEED.game.tsv" "$S/tele-$SEED.d2d-$t.tsv"
