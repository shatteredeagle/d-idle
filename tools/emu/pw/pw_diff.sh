#!/bin/bash
# prwarp (worker host): the warp arrival oracle against one build of test_warp_arrival.
#   pw_diff.sh save TAG          copy build-pw's test_warp_arrival to ~/d2d-lab/pw/bin/TAG
#   pw_diff.sh synth TAG SEED N  gen --warp (cached per SEED N), game.exe's answers (cached), d2d's from bin/TAG, diff
set -u
here=$(cd "$(dirname "$0")" && pwd)
W=$HOME/d2d-lab/wt-pw; O=$HOME/d2d-lab/pw; mkdir -p $O/bin
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
case $1 in
save) cp $HOME/d2d-lab/build-pw/tests/test_warp_arrival $O/bin/$2 && echo saved $2 ;;
synth)
  tag=$2; seed=$3; n=$4; D=$O/synth-$seed-$n; mkdir -p $D
  cd $W/tools/emu
  [ -s $D/cases.tsv ] || nice -n 19 $PY sk_tele_spot.py gen --warp $n $seed > $D/cases.tsv
  [ -s $D/game.tsv ] || nice -n 19 $PY sk_tele_spot.py warp $D/cases.tsv > $D/game.tsv
  $O/bin/$tag --warp-arrival $D/cases.tsv | grep -E '^[0-9]+ ' > $D/$tag.tsv
  echo "== $tag seed $seed, $n cases: game.exe lands $(awk '$2==1' $D/game.tsv | wc -l), refuses $(awk '$2==0' $D/game.tsv | wc -l)"
  nice -n 19 $PY sk_tele_spot.py diff $D/game.tsv $D/$tag.tsv $D/cases.tsv; echo "diff rc=$?"
  ;;
esac
