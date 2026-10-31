#!/bin/bash
# ON THE WORKER: control = unpatched lab5 source built with the same recipe/flags as the patched build (/tmp/prmeasure-ctl-b).
OUT=${1:-$HOME/d2d-lab/prmeasure}; HERE=$(dirname "$(readlink -f "$0")"); B=/tmp/prmeasure-ctl-b/apps/d2d/d2d
for S in 1 2 3 4; do for LC in "1 2" "4 5" "8 9"; do echo "control $B $LC $S"; done; done | xargs -P3 -L1 bash -c 'T=$0; B=$1; L=$2; C=$3; S=$4; \
  $HOME/d2d-lab/d2dslot.sh run -w 3000 -- bash '"$HERE"'/runone.sh $T $B $L $C $S '"$OUT"
echo ALLDONE > "$OUT/done-ctl"
