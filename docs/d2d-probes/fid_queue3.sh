#!/bin/bash
cd ~/d2d-lab/fidelity || exit 1
while pgrep -f "[f]id_queue2.sh" > /dev/null; do sleep 10; done
python3 -u fid_hit.py out/hit6.jsonl 6 > out/hit6.log 2>&1
python3 -c "import fid; fid.stop()"
