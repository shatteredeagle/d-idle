#!/bin/bash
# decomp5 (worker host): make the local candidate branch local/fid-load-gold-clamp in the scratch tree and build test_game
set -eu
T=$HOME/d2d-lab/wt-decomp5; H=$(cd "$(dirname "$0")" && pwd)
cd $T
git checkout -q -- tests/game/test_game.cpp
git checkout -q -b local/fid-load-gold-clamp 1453f853 2>&1 || git checkout -q local/fid-load-gold-clamp
D5_TREE=$T python3 $H/d5_patch_loadgold.py
D5_TREE=$T python3 $H/d5_mkgold.py
cp $H/d5_fl_gold.py.src $T/tools/emu/fl_gold.py
git status --short
bash $H/d5_build.sh
