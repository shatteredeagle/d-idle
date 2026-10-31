#!/bin/bash
# ON THE WORKER: interleaved 3-arm rerun, one slot at a time, gated on 1-min load < 6 (total wait budget 40 min, then run anyway).
# Writes OUT/load.csv (arm,cb,seed,load1_before,load1_after) next to the d2dcasts2 outputs.
OUT=${1:-$HOME/d2d-lab/prmeasure2}; HERE=$(dirname "$(readlink -f "$0")"); mkdir -p "$OUT"
declare -A BIN=([before]=$HOME/d2d-lab/fork/bin/d2d.lab5 [control]=/tmp/prmeasure-ctl-b/apps/d2d/d2d [after]=/tmp/prbuild-d2d.new)
budget=$((40*60)); : > "$OUT/load.csv"
for LC in "1 2" "4 5"; do set -- $LC; L=$1; C=$2
 for S in 1 2 3 4 5 6 7 8; do for T in before control after; do
  while [ $budget -gt 0 ] && [ "$(cut -d' ' -f1 /proc/loadavg | cut -d. -f1)" -ge 6 ]; do sleep 10; budget=$((budget-10)); done
  l0=$(cut -d' ' -f1 /proc/loadavg)
  $HOME/d2d-lab/d2dslot.sh run -w 3000 -- bash "$HERE/runone.sh" $T ${BIN[$T]} $L $C $S "$OUT"
  echo "$T,$L,$S,$l0,$(cut -d' ' -f1 /proc/loadavg)" >> "$OUT/load.csv"
 done; done
done
echo ALLDONE > "$OUT/done"
