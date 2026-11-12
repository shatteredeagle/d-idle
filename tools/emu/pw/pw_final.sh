#!/bin/bash
# prwarp (worker host): rewrite the branch with the final messages (c3.sh), build each commit alone (all targets), the test, then the checks.
here=$(cd "$(dirname "$0")" && pwd)
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
bash $here/c3.sh || exit 1
T=$HOME/d2d-lab/wt-pw
c2=$(git -C $T rev-parse HEAD); c1=$(git -C $T rev-parse HEAD~1)
bash $here/pw_build.sh $c1 || exit 1
echo "--- test at the oracle commit"; $HOME/d2d-lab/build-pw/tests/test_warp_arrival; echo "rc=$?"
bash $here/pw_build.sh $c2 || exit 1
echo "--- test at the claim commit"; $HOME/d2d-lab/build-pw/tests/test_warp_arrival 2>&1 | tail -3; echo "rc=${PIPESTATUS[0]}"
bash $here/pw_diff.sh save claim2
bash $here/pw_diff.sh synth claim2 1 20000
bash $here/pw_checks.sh
echo FINALDONE
