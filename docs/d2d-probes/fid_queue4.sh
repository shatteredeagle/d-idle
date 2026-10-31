#!/bin/bash
cd ~/d2d-lab/fidelity || exit 1
while pgrep -f "[f]id_queue3.sh" > /dev/null; do sleep 10; done
python3 -u fid_stam.py > out/stam.log 2>&1
