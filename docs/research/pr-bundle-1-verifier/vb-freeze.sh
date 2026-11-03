#!/bin/bash
# vb-freeze.sh N SEED : freeze_rules differential (existing upstream oracle) on the bundle head, own seed
H=~/d2d-lab/verifybundle/head; O=~/d2d-lab/verifybundle/out-head; mkdir -p $O
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY="nice -n 19 $HOME/d2d-lab/pr/walk2-venv/bin/python"
cd $H/tools/emu || exit 2
$PY freeze_rules.py gen $1 $2 > $O/fr.cases && $PY freeze_rules.py game $O/fr.cases > $O/fr.game 2> $O/fr.game.err || { echo game rc $?; tail -3 $O/fr.game.err; exit 1; }
nice -n 19 ~/d2d-lab/verifybundle/build-head/tests/test_freeze_rules --freeze-rules $O/fr.cases > $O/fr.d2d 2> $O/fr.d2d.err; echo d2d rc $?
$PY freeze_rules.py compare $O/fr.cases $O/fr.game $O/fr.d2d 2>&1 | tail -5
