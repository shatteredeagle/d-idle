#!/bin/bash
# decomp7 (worker host): TASK-75 xp grid, game.exe FUN_0057e2f0 vs d2d rules::kill_exp on the scratch tree.
HERE=$(cd "$(dirname "$0")" && pwd)
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp6 B=$HOME/d2d-lab/build-decomp6 O=$HOME/d2d-lab/d7; mkdir -p $O
cp $HERE/xp_award.py $T/tools/emu/; cd $T/tools/emu
git -C $T log --oneline -1
g++ -std=c++23 -O1 $(find $T/components -maxdepth 2 -type d -printf "-I%p ") $HERE/d7_kill_exp.cpp -o $O/kexp 2>&1 | head -5
$O/kexp > $O/xp.d; wc -l < $O/xp.d
uv run python xp_award.py grid 2>/dev/null > $O/xp.g; wc -l < $O/xp.g
echo "differ: $(diff $O/xp.g $O/xp.d | grep -c '^<')"
echo "== kill_exp.py 100000 5 (upstream's own random oracle, same generator as test_combat)"
uv run python kill_exp.py 100000 5 2>/dev/null > $O/ke.g; wc -l < $O/ke.g
$O/kexp stdin < $O/ke.g > $O/ke.d
echo "differ: $(diff $O/ke.g $O/ke.d | grep -c '^<')"
