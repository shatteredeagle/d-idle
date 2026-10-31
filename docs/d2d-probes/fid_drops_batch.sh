#!/bin/bash
# usage: fid_drops_batch.sh OUT.jsonl FIRSTSEED LASTSEED   (runs fid_sweep.py once per seed, clvl cycles 10..14, Sorceress, levels BM CP SF Den DW)
cd ~/d2d-lab/fidelity || exit 1
for s in $(seq $2 $3); do
  c=$((10 + s % 5))
  python3 fid_sweep.py "$1" $s $c 4 2 3 4 8 5 >> "${1%.jsonl}.log" 2>&1
done
python3 -c "import fid; fid.stop()"
