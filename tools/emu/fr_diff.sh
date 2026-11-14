#!/bin/sh
# fidrevive (worker host): see docs/research/fid-revive-refill.md section 6; run through the worker host runner from the synced scratch clone of the fork branch.
export D2_BUILD=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/upstream/build D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq
T=${FR_TREE:?the synced scratch clone of the fork branch}; B=$HOME/d2d-lab/build-fidrevive; O=/tmp/fr_out; mkdir -p $O
cd $T/tools/emu
[ -f $O/cases.tsv ] || nice -n 19 uv run python fr_revive.py gen 400 > $O/cases.tsv
[ -f $O/game.tsv ] || nice -n 19 uv run python fr_revive.py game $O/cases.tsv > $O/game.tsv
tag=${1:-after}
(cd $B && nice -n 19 ./tests/test_game --revive-stats $O/cases.tsv 2> $O/d2d.err | grep -E "^[A-Za-z_][A-Za-z_0-9]* -?[0-9]+( -?[0-9]+){5}\$" > $O/d2d.$tag.tsv)
echo "cases $(wc -l < $O/cases.tsv) game $(wc -l < $O/game.tsv) d2d $(wc -l < $O/d2d.$tag.tsv)"
echo "mismatching cases ($tag): $(diff $O/game.tsv $O/d2d.$tag.tsv | grep -c '^<')"
diff $O/game.tsv $O/d2d.$tag.tsv | head -${SHOW:-6}
