#!/bin/bash
# prwarp (worker host): the upstream checks of the branch head in wt-pw (built in build-pw): pr_guard, presubmit --quick, ctest.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
cd $HOME/d2d-lab/wt-pw || exit 1
git log --oneline -3
echo "=== pr_guard"; python3 tools/pr_guard.py upstream/main HEAD; echo "rc=$?"
echo "=== pr_guard --sorted"; python3 tools/pr_guard.py --sorted upstream/main HEAD 2>&1 | tail -3; echo "rc=$?"
echo "=== presubmit --quick"; timeout 1500 python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -vE '^ok' | tail -12; echo "rc=${PIPESTATUS[0]}"
echo "=== ctest"; nice -n 19 ctest --test-dir $HOME/d2d-lab/build-pw -j8 --timeout 600 2>&1 | tail -12; echo "ctest rc=${PIPESTATUS[0]}"
echo CHECKSDONE
