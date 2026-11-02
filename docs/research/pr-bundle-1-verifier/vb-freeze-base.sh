#!/bin/bash
# vb-freeze-base.sh : the d2d side of the freeze cases on the base build (bundle's test file copied in), compare
O=~/d2d-lab/verifybundle/out-head; B=~/d2d-lab/verifybundle/build-base
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
nice -n 19 $B/tests/test_freeze_rules --freeze-rules $O/fr.cases > $O/fr.base.d2d 2> $O/fr.base.err; echo base d2d rc $?
cd ~/d2d-lab/verifybundle/head/tools/emu && nice -n 19 $HOME/d2d-lab/pr/walk2-venv/bin/python freeze_rules.py compare $O/fr.cases $O/fr.game $O/fr.base.d2d 2>&1 | tail -2
