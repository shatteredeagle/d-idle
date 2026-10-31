#!/bin/bash
cd ~/d2d-lab/fidelity || exit 1
while pgrep -f "[f]id_queue5.sh" > /dev/null; do sleep 10; done
python3 -u fid_hit.py out/hit7.jsonl 7 > out/hit7.log 2>&1
python3 -c "import fid; fid.stop()"
