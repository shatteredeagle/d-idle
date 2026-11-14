#!/bin/bash
# the load differential on the worker host: game.exe (fl_load.py) against d2d's World::enter (test_game --load-stats)
# usage: fo_diff.sh [N random cases] ; BASE=1 swaps setup.cpp for the parent's and rebuilds first
set -u
T=${FO_TREE:?set FO_TREE to the synced scratch checkout of the fork branch}; B=${FO_BUILD:-$HOME/d2d-lab/build-fidoracle}
export D2_BUILD=${FO_MPQCAT_BUILD:-${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build} D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
O=/tmp/fo_out; mkdir -p $O
cd $T/tools/emu
[ -f $O/cases.tsv ] || nice -n 19 uv run python fl_load.py gen ${1:-400} > $O/cases.tsv
[ -f $O/game.tsv ] || nice -n 19 uv run python fl_load.py game $O/cases.tsv > $O/game.tsv
if [ "${BASE:-0}" = 1 ]; then
  cp $T/components/game/world/setup.cpp $O/setup.cpp.new
  git -C $T show ${REV:-da2d96cf}:components/game/world/setup.cpp > $T/components/game/world/setup.cpp
  nice -n 19 ninja -C $B -j2 test_game > $B.base.log 2>&1; echo "base build rc=$?"
fi
(cd $B && nice -n 19 ./tests/test_game --load-stats $O/cases.tsv 2> $O/d2d.err | grep -E "^[A-Za-z_][A-Za-z_0-9]* -?[0-9]+( -?[0-9]+){5}\$" > $O/d2d${BASE:+.${REV:-base}}.tsv)
[ "${BASE:-0}" = 1 ] && { cp $O/setup.cpp.new $T/components/game/world/setup.cpp; }
N=$(wc -l < $O/cases.tsv)
F=$O/d2d${BASE:+.${REV:-base}}.tsv
echo "cases $N game $(wc -l < $O/game.tsv) d2d $(wc -l < $F)"
echo "mismatching cases: $(diff $O/game.tsv $F | grep -c '^<')"
diff $O/game.tsv $F | head -${SHOW:-12}
