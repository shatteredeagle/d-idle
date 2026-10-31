#!/bin/bash
cd ~/d2d-lab/fidelity || exit 1
while pgrep -f "[f]id_queue4.sh" > /dev/null; do sleep 10; done
for s in 51 52 53; do python3 -u fid_sweep.py out/drops3.jsonl $s $((10 + s % 5)) 4 4 >> out/drops3.log 2>&1; done
python3 -c "import fid; fid.stop()"
