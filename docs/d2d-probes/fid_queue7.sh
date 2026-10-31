#!/bin/bash
cd ~/d2d-lab/fidelity || exit 1
while pgrep -f "[f]id_queue6.sh" > /dev/null; do sleep 5; done
python3 -u fid_hit.py out/hit8.jsonl 8 > out/hit8.log 2>&1
python3 -c "import fid; fid.stop()"
