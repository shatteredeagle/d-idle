#!/bin/bash
# decomp5 (worker host): get-hit decision (FUN_0057cee0) and drawn-frame (FUN_004b1280) differentials against test_combat on current main
T=$HOME/d2d-lab/wt-decomp5; B=$HOME/d2d-lab/build-decomp5; O=$HOME/d2d-lab/d5/hit
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe D2_BUILD=$B
H=$HOME/labrun/herdr/code/d2lod-wt/decomp5/tools/emu
cp $H/hit_rule.py $H/hit_render.py $T/tools/emu/; cd $T/tools/emu
nice -n 19 uv run python hit_rule.py gen ${1:-100000} 7 > $O/r_cases.tsv
nice -n 19 uv run python hit_rule.py game $O/r_cases.tsv > $O/r_game.tsv
(cd $B && nice -n 19 tests/test_combat --get-hit < $O/r_cases.tsv 2>/dev/null | grep -E '^[0-9]+ ' > $O/r_d2d.tsv)
echo "rule: cases $(grep -vc '^#' $O/r_cases.tsv) game $(wc -l < $O/r_game.tsv) d2d $(wc -l < $O/r_d2d.tsv)"
nice -n 19 uv run python hit_rule.py compare $O/r_game.tsv $O/r_d2d.tsv | tail -4
# drawn frames: kind-1 cases of the length run
grep -E '^#|^[0-9]+ 1 ' $O/cases.tsv > $O/f_cases.tsv
nice -n 19 uv run python hit_render.py game $O/f_cases.tsv > $O/f_game.tsv
(cd $B && nice -n 19 tests/test_combat --get-hit-frames < $O/f_cases.tsv 2>/dev/null | grep -E '^[0-9]+ ' > $O/f_d2d.tsv)
echo "frames: cases $(grep -vc '^#' $O/f_cases.tsv) game $(wc -l < $O/f_game.tsv) d2d $(wc -l < $O/f_d2d.tsv)"
nice -n 19 uv run python hit_render.py compare $O/f_game.tsv $O/f_d2d.tsv $O/f_cases.tsv 1 | tail -4
