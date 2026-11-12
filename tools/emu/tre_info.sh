#!/bin/bash
# treasure card: which sources the d2d drop-dump build under test was made from (worker host, read-only).
F=$HOME/d2d-lab/fork
W=$F/wt-pristine
git -C $W log --oneline -1
git -C $W branch --show-current
git -C $W status --short | wc -l
git -C $W diff --stat 72132b99 -- components/rules/drops.hpp components/game/loot.hpp tools/drop-dump | tail -2
git -C $W log --oneline -1 72132b99 | head -1
git -C $W merge-base --is-ancestor 72132b99 HEAD && echo "72132b99 is an ancestor of the pristine head"
ls -la --time-style=long-iso $F/build-pristine/tools/drop-dump/drop-dump
grep -m3 "CMAKE_HOME_DIRECTORY\|CMAKE_BUILD_TYPE" $F/build-pristine/CMakeCache.txt
git -C $HOME/d2d-lab/pr/skmana log -1 --format=%h
git -C $F merge-base HEAD 72132b99 2>&1 | head -1
git -C $F log --oneline -1 72132b99 2>&1 | head -1
