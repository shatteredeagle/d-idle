#!/bin/bash
# decomp7 (worker host): TASK-84 monster missile record, game.exe vs the real d2d build (upstream main scratch tree).
HERE=$(cd "$(dirname "$0")" && pwd)
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_MPQ_DIR:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq} D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp6 B=$HOME/d2d-lab/build-decomp6 O=$HOME/d2d-lab/d7/mm; mkdir -p $O
N=${1:-60000} S=${2:-20261109}
cd $T/tools/emu; git -C $T log --oneline -1
uv run python mm_diff.py game $HOME/mmtab $N $S $O 2>&1 | grep -v WARN | tail -4
$B/tests/test_monster_missile --monster-missile $O/cases.txt > $O/d2d.out 2>&1; echo "d2d rc=$?"
uv run python mm_diff.py cmp $O $O/d2d.out 2>&1 | grep -v WARN | tail -12
