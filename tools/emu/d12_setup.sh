#!/bin/bash
# decomp12 (worker host): scratch tree of upstream main + a Release build of libgame for the d12_* probes.
export PATH=$HOME/.local/bin:$PATH
R=$HOME/d2d-lab/wt-decomp8; T=$HOME/d2d-lab/wt-decomp12; B=$HOME/d2d-lab/build-decomp12
git -C $R fetch -q upstream 2>&1 | tail -1
[ -d $T ] || git -C $R worktree add --detach $T upstream/main
git -C $T log --oneline -1
[ -f $B/CMakeCache.txt ] || nice -n 19 cmake -S $T -B $B -DCMAKE_BUILD_TYPE=Release > $HOME/d2d-lab/build-decomp12.cmake.log 2>&1
tail -2 $HOME/d2d-lab/build-decomp12.cmake.log
nice -n 19 cmake --build $B -j 6 --target game 2>&1 | tail -3
ls -l $B/components/game/libgame.a
