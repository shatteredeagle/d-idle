#!/bin/bash
# prprep (worker host): Cold Enchanted length differential on the rebased candidate.   pp_cold.sh after|before SEED N
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold; B=$HOME/d2d-lab/build-pp-cold; O=$HOME/d2d-lab/pp-cold; mkdir -p $O
mode=$1; seed=${2:-1}; n=${3:-100000}
cd $T
if [ $mode = before ]; then git checkout -q HEAD~1 || exit 1; fi
bash $HERE/pp_build.sh cold test_umod_stats mpq-cat | tail -2
git log --oneline -1
cd $T/tools/emu
[ -s $O/game-$seed-$n.tsv ] || uv run python umod_stats.py $n $seed > $O/game-$seed-$n.tsv 2>$O/game.err
echo "game cases: $(wc -l < $O/game-$seed-$n.tsv)"
$B/tests/test_umod_stats --umod-stats $O/game-$seed-$n.tsv > $O/$mode.d2d 2>$O/$mode.err; echo "d2d rc=$?"
uv run python umod_stats.py --compare $O/game-$seed-$n.tsv $O/$mode.d2d 2>&1 | grep -v WARNING | tail -25 | tee $O/$mode.cmp
cd $T; if [ $mode = before ]; then git checkout -q fid/cold-enchanted-length-r1; fi
