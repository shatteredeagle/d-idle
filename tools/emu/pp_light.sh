#!/bin/bash
# prprep (worker host): Lightning one-unit-a-frame differential on the rebased candidate, after (HEAD) and before (HEAD~1).
#   pp_light.sh after|before SEED N
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-pp-light; B=$HOME/d2d-lab/build-pp-light; O=$HOME/d2d-lab/pp-light; mkdir -p $O
HERE=$(cd "$(dirname "$0")" && pwd)
mode=$1; seed=${2:-1}; n=${3:-400}
cd $T
if [ $mode = before ]; then git checkout -q HEAD~1 || exit 1; bash $HERE/pp_build.sh light test_light_pierce mpq-cat | tail -2; fi
git log --oneline -1
cd $T/tools/emu
$B/tests/test_light_pierce --light-hit $seed $n > $O/$mode.d 2>$O/$mode.err; echo "d2d rc=$? casts $(grep -c '^T' $O/$mode.d)"
uv run python light_pierce.py $O/$mode.d > $O/$mode.g 2>$O/$mode.gerr; echo "game rc=$? casts $(grep -c '^T' $O/$mode.g)"
grep -E '^[HT]' $O/$mode.d | diff $O/$mode.g - > $O/$mode.diff; echo "$mode seed $seed n $n differing lines: $(grep -c '^[<>]' $O/$mode.diff)  T-lines differing: $(grep -c '^[<>] T' $O/$mode.diff)"
cd $T; if [ $mode = before ]; then git checkout -q fid/missile-last-unit-r1; fi
