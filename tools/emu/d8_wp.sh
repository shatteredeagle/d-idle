#!/bin/bash
# decomp8 (worker host): TASK-95 waypoint refusal window, game.exe (wp_refuse.py game) against the real World::apply
# (test_game --waypoint-refuse) on upstream main. usage: d8_wp.sh [seed]
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp8 B=$HOME/d2d-lab/build-decomp8 O=$HOME/d2d-lab/d8; mkdir -p $O
cd $T/tools/emu; git -C $T log --oneline -1
for s in ${1:-7} 271828; do
  uv run python wp_refuse.py gen 100000 $s --solo > $O/wp$s.txt 2>/dev/null
  nice -n 19 uv run python wp_refuse.py game $O/wp$s.txt 2>/dev/null > $O/wp$s.g
  $B/tests/test_game --waypoint-refuse $O/wp$s.txt | grep -E '^[0-9]+ ' > $O/wp$s.d
  echo "seed $s: cases $(wc -l < $O/wp$s.txt) game $(wc -l < $O/wp$s.g) d2d $(wc -l < $O/wp$s.d) differing $(diff $O/wp$s.g $O/wp$s.d | grep -c '^<')"
  md5sum $O/wp$s.g $O/wp$s.d
done
