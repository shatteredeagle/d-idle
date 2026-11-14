#!/bin/bash
# prprep (worker host): rebuild the Lightning candidate's neighbours and run the gates.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
bash $HERE/pp_build.sh light test_light_pierce test_next_hit test_game_missiles test_monsters test_combat test_game test_missile_hit | tail -2
B=$HOME/d2d-lab/build-pp-light
for t in test_game_missiles test_monsters test_combat test_game; do timeout 900 $B/tests/$t >/tmp/pp_$t.out 2>&1; echo "$t rc=$?"; done
bash $HERE/pp_light.sh after 1 400; bash $HERE/pp_light.sh after 7 400
cd $HOME/d2d-lab/wt-pp-light/tools/emu
$B/tests/test_next_hit --next-hit 1 400 > /tmp/pp_nh.d 2>/dev/null; uv run python next_hit.py /tmp/pp_nh.d > /tmp/pp_nh.g; echo "next_hit differing: $(grep -E '^[HT]' /tmp/pp_nh.d | diff /tmp/pp_nh.g - | wc -l) of $(grep -c '^T' /tmp/pp_nh.d)"
