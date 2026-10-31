#!/bin/bash
cd ~/d2d-lab/fidelity || exit 1
python3 -u fid_misc.py > out/misc.log 2>&1
python3 -u fid_hit.py out/hit.jsonl 6 > out/hit5.log 2>&1
for s in 41 42 43 44 45 46; do python3 -u fid_sweep.py out/lowxp.jsonl $s $((4 + s % 3)) 4 2 3 8 >> out/lowxp.log 2>&1; done
python3 -c "import fid; fid.stop()"
