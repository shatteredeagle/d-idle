#!/bin/bash
# verifyb2 (the worker host): the d2d side of the mana check, base vs head, on a game.exe case set.
#   vb2_mana_d2d.sh SEED TREE...   (trees built by vb2_build.sh; cases from vb2_mana_run.sh)
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-pa-verifyb2}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
SEED=$1; shift
for t in "$@"; do
  [ -x "$HOME/d2d-lab/vb2/bin/vb2_mana_probe-$t" ] || bash "$here/vb2_probe_build.sh" "$t" vb2_mana_probe
  nice -n 19 "$HOME/d2d-lab/vb2/bin/vb2_mana_probe-$t" "$S/mana-$SEED.cases.tsv" > "$S/mana-$SEED.d2d-$t.tsv"
  echo "== $t vs game.exe"
  bash "$here/run-py.sh" vb2_mana.py cmp "$S/mana-$SEED.cases.tsv" "$S/mana-$SEED.game.tsv" "$S/mana-$SEED.d2d-$t.tsv" "$S/skills.tsv"
done
