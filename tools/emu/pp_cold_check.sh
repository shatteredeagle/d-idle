#!/bin/bash
# prprep (worker host): re-run the Cold Enchanted candidate's gates and print one summary (the finish script's stdout was lost).
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-pp-cold; B=$HOME/d2d-lab/build-pp-cold; O=$HOME/d2d-lab/pp-cold
cd $T; git log --oneline -1
for t in test_boss_cold_length test_monsters test_game; do timeout 900 $B/tests/$t >/tmp/pp_$t.out 2>&1; echo "$t rc=$?"; done
cd tools/emu
uv run python boss_level.py compare $O/bl.tsv $O/bl.g $O/bl.d 2>&1 | grep -v WARNING | tail -4
cd $T; python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
git ls-remote forgejo fid/cold-enchanted-length-r1 | cut -c1-12
