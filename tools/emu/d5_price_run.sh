#!/bin/bash
# decomp5 (worker host): re-run of the price differentials of fid-vendor-price-trace.md on the current scratch tree of upstream main
T=$HOME/d2d-lab/wt-decomp5; B=$HOME/d2d-lab/build-decomp5; O=$HOME/d2d-lab/d5/price
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe D2_BUILD=$B
git -C $T log --oneline -1
for run in "gen 100000 7" "gen 100000 1" "edge 20000 3" "tomes"; do
  echo "== $run"; bash $(cd "$(dirname "$0")" && pwd)/fp_price_runs.sh $T $B $O/"${run// /_}" $run
done
