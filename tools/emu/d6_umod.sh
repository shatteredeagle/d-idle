#!/bin/bash
# decomp6 (worker host): the pr-mods evidence of monster-mods-trace.md (umod_stats.py, 157,618 cases) on the scratch tree of upstream main
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
git -C $T log --oneline -1
cp $HERE/umod_*.py $HERE/umod_stats_harness.cpp $T/tools/emu/
cd $T/tools/emu
mkdir -p $O/umod
uv run python umod_stats.py 100000 1 > $O/umod/game.tsv 2>$O/umod/game.err; echo "game rc=$? lines $(wc -l < $O/umod/game.tsv)"
g++ -O2 -std=c++23 $(for d in ../../components/*/; do echo -n "-I$d "; done) -o $O/umod/harness umod_stats_harness.cpp 2>&1 | head -20; echo "build rc=${PIPESTATUS[0]}"
$O/umod/harness < $O/umod/game.tsv > $O/umod/d2d.tsv; echo "d2d lines $(wc -l < $O/umod/d2d.tsv)"
uv run python umod_stats.py --compare $O/umod/game.tsv $O/umod/d2d.tsv 2>&1 | grep -v WARNING | tail -40
