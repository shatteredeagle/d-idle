#!/bin/bash
# decomp8 (worker host): TASK-95 town portal read from the pack, game.exe (tome_use.py game) against the real
# d2d (test_game --tome-use) on upstream main. usage: d8_tome.sh [seed...]
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp8 B=$HOME/d2d-lab/build-decomp8 O=$HOME/d2d-lab/d8; mkdir -p $O
cd $T/tools/emu; git -C $T log --oneline -1
for s in ${@:-7 271828}; do
  uv run python tome_use.py gen 100000 $s > $O/tu$s.tsv 2>/dev/null
  nice -n 19 uv run python tome_use.py game $O/tu$s.tsv 2>/dev/null > $O/tu$s.g
  $B/tests/test_game --tome-use $O/tu$s.tsv | grep -E '^[0-9]+ ' > $O/tu$s.d
  echo "seed $s: cases $(wc -l < $O/tu$s.tsv) game $(wc -l < $O/tu$s.g) d2d $(wc -l < $O/tu$s.d) differing $(diff $O/tu$s.g $O/tu$s.d | grep -c '^<')"
  diff $O/tu$s.g $O/tu$s.d | head -6
done
