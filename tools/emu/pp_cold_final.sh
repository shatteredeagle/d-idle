#!/bin/bash
# prprep (worker host): final run of the Cold Enchanted branch after the tidy and README fixes.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold; B=$HOME/d2d-lab/build-pp-cold; O=$HOME/d2d-lab/pp-cold
cd $T; git rev-parse HEAD
bash $HERE/pp_build.sh cold test_boss_cold_length test_umod_stats test_monsters test_game mpq-cat | tail -2
for t in test_boss_cold_length test_monsters test_game; do timeout 900 $B/tests/$t >/tmp/pp_$t.out 2>&1; echo "$t rc=$?"; done
cd tools/emu
$B/tests/test_umod_stats --umod-stats $O/game-1-100000.tsv > $O/final.d2d && uv run python umod_stats.py --compare $O/game-1-100000.tsv $O/final.d2d 2>&1 | grep -v WARNING > $O/final.full
python3 pp_cold_count.py $O/before.full $O/final.full
cd $T
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $O/presubmit2.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $O/presubmit2.txt | tail -8 | cut -c1-260
