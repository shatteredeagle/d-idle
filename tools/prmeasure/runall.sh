#!/bin/bash
# ON THE WORKER: 24 runs (before/after x CB 1,4,8 x seeds 1-4), 3 slots at once, interleaved so load is the same for A and B.
OUT=${1:-$HOME/d2d-lab/prmeasure}; HERE=$(dirname "$(readlink -f "$0")")
BEFORE=$HOME/d2d-lab/fork/bin/d2d.lab5; AFTER=/tmp/prbuild-d2d.new
for S in 1 2 3 4; do for LC in "1 2" "4 5" "8 9"; do for T in before after; do
  [ $T = before ] && B=$BEFORE || B=$AFTER
  echo "$T $B $LC $S"
done; done; done | xargs -P3 -L1 bash -c 'T=$0; B=$1; L=$2; C=$3; S=$4; \
  $HOME/d2d-lab/d2dslot.sh run -w 3000 -- bash '"$HERE"'/runone.sh $T $B $L $C $S '"$OUT"
echo ALLDONE > "$OUT/done"
