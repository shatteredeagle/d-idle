#!/bin/bash
# decomp8 (worker host): scratch tree of upstream main and a build of the test binaries the fid cards need.
export PATH=$HOME/.local/bin:$PATH
R=$HOME/d2d-lab/wt-decomp6; T=$HOME/d2d-lab/wt-decomp8; B=$HOME/d2d-lab/build-decomp8
[ -d $T ] || git -C $R worktree add --detach $T upstream/main
git -C $T log --oneline -1
grep -E "CMAKE_BUILD_TYPE|CMAKE_GENERATOR:" $HOME/d2d-lab/build-decomp6/CMakeCache.txt
[ -f $B/CMakeCache.txt ] || nice -n 19 cmake -S $T -B $B -DCMAKE_BUILD_TYPE=Release > $HOME/d2d-lab/build-decomp8.cmake.log 2>&1
tail -2 $HOME/d2d-lab/build-decomp8.cmake.log
