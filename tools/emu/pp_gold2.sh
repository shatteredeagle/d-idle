#!/bin/bash
# prprep (worker host): restore the gold clamp claim (the first script applied it from a subdirectory) and run test_game with it.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-gold; B=$HOME/d2d-lab/build-pp-gold; O=$HOME/d2d-lab/pp-gold
cd $T && git apply $O/claim.patch && git status --short | head -3
bash $HERE/pp_build.sh gold test_game | tail -2
cd $B && timeout 900 tests/test_game > $O/tg2.out 2>&1; echo "test_game with the claim rc=$?"
cd $T; git log --format='%h %an %cn' -1
