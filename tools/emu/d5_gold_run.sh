#!/bin/bash
# decomp5 (worker host): gold clamp oracle: limits of FUN_00622e70/623460 and the load clamp vs d2d. usage: d5_gold_run.sh [limits|load]
T=$HOME/d2d-lab/wt-decomp5; B=$HOME/d2d-lab/build-decomp5; O=$HOME/d2d-lab/d5/gold
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe D2_BUILD=$B
mkdir -p $O; H=$(cd "$(dirname "$0")" && pwd)
cd $T/tools/emu
case ${1:-load} in
 limits) nice -n 19 uv run python fl_gold.py limits > $O/limits.txt; echo "levels $(wc -l < $O/limits.txt) diff $(grep -c DIFF $O/limits.txt)"; head -3 $O/limits.txt; tail -2 $O/limits.txt ;;
 load) nice -n 19 uv run python fl_gold.py gen > $O/cases.tsv
   nice -n 19 uv run python fl_gold.py game $O/cases.tsv > $O/game.tsv
   (cd $B && nice -n 19 tests/test_game --load-gold $O/cases.tsv 2>/dev/null | grep -E '^[A-Za-z_][A-Za-z_0-9]* -?[0-9]+( -?[0-9]+){1}$' > $O/d2d.tsv)
   echo "cases $(wc -l < $O/cases.tsv) game $(wc -l < $O/game.tsv) d2d $(wc -l < $O/d2d.tsv) mismatches $(diff $O/game.tsv $O/d2d.tsv | grep -c '^<')"
   diff $O/game.tsv $O/d2d.tsv | head -8 ;;
esac
