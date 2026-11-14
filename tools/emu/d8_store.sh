#!/bin/bash
# decomp8 (worker host): TASK-77 vendor stock (store.py, FUN_00576980) and merc offers (hire.py) from game.exe's code against
# the real drlg-dump `store` / `hire` modes of upstream main. usage: d8_store.sh FIRST-LAST
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp8 B=$HOME/d2d-lab/build-decomp8 O=$HOME/d2d-lab/d8/store; mkdir -p $O
cd $T/tools/emu; git -C $T log --oneline -1
for what in store hire; do
  nice -n 19 uv run python $what.py $1 > $O/$what.g 2> $O/$what.g.err
  $B/tools/drlg-dump/drlg-dump $D2_MPQ_DIR $1 0 $what 2>/dev/null | grep -v -E '^20[0-9-]+T' > $O/$what.d
  echo "$what: game $(wc -l < $O/$what.g) lines, d2d $(wc -l < $O/$what.d) lines, differing $(diff $O/$what.g $O/$what.d | grep -c '^<')"
  diff $O/$what.g $O/$what.d | head -6 | cut -c1-200
done
head -3 $O/store.g | cut -c1-200
