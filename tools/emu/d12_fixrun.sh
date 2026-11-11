#!/bin/bash
# decomp12 (worker host): the fix candidate. Branch fid/warp-arrival-r1 of the fork in its own worktree off upstream main,
# patched by d12_fix.py (committed by hand afterwards), built, then the levels differential with the probe built -DD12_FIXED.
export PATH=$HOME/.local/bin:$PATH
here=$(cd "$(dirname "$0")" && pwd)
R=$HOME/d2d-lab/wt-decomp12; T=$HOME/d2d-lab/wt-decomp12fix; B=$HOME/d2d-lab/build-decomp12fix
if [ ! -d $T ]; then
  git -C $R worktree add -b fid/warp-arrival-r1 $T 8d7b89f1 || exit 1
  python3 $here/d12_fix.py $T || exit 1
fi
git -C $T log --oneline -1
[ -f $B/CMakeCache.txt ] || nice -n 19 cmake -S $T -B $B -DCMAKE_BUILD_TYPE=Release > $B.cmake.log 2>&1
nice -n 19 cmake --build $B -j 6 --target game 2>&1 | tail -2
D12_SRC=$T D12_BUILD=$B D12_DEFS=-DD12_FIXED D12_SUFFIX=_fixed bash $here/d12_build.sh || exit 1
export D12_PROBE=$HOME/d2d-lab/d12/d12_arrive_probe_fixed
for s in 5 82 7; do bash $here/d12_run.sh levels $s 2>&1 | grep -E "^seed|^==|^cases|both|game "; done
