#!/bin/bash
# sequential queue of the remaining probes (one d2d at a time, every start waits for load <= 45; see fid.safety)
cd ~/d2d-lab/fidelity || exit 1
python3 -u fid_potion.py out/potion_sorc_vit150.json 4 150 > out/potion_vit150.log 2>&1
python3 -u fid_aggro.py out/aggro.jsonl 3 2 3 4 8 > out/aggro3.log 2>&1
python3 -u fid_aggro.py out/aggro.jsonl 4 2 3 4 8 > out/aggro4.log 2>&1
for s in 31 32 33 34 35 36 37 38; do python3 -u fid_sweep.py out/drops2.jsonl $s $((10 + s % 5)) 4 2 8 >> out/drops2.log 2>&1; done
python3 -c "import fid; fid.stop()"
# low-clvl XP comparison with the live soak table (docs/soak-analysis.md section 2): clvl 4, 5, 6
for s in 41 42 43 44 45 46; do python3 -u fid_sweep.py out/lowxp.jsonl $s $((4 + s % 3)) 4 2 3 8 >> out/lowxp.log 2>&1; done
python3 -c "import fid; fid.stop()"
python3 -u fid_hit.py out/hit.jsonl 5 > out/hit5.log 2>&1
python3 -u fid_misc.py > out/misc.log 2>&1
