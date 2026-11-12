#!/bin/bash
# UNIQMODS: umod_spawn.py over Act 1 and Act 2 outdoor areas and the three difficulties (V run: game.exe's own room population).
# Output: $D2_SCRATCH/um_d<diff>_a<area>.txt, one per (difficulty, area).
set -u
out=${D2_SCRATCH:-/tmp}
for d in 0 1 2; do
	for a in 3 4 5 8 17 41 43 44; do
		bash tools/emu/run-py.sh umod_spawn.py 1-8 $a $d > "$out/um_d${d}_a${a}.txt" 2>&1
		echo "d=$d area=$a lines=$(wc -l < "$out/um_d${d}_a${a}.txt")"
	done
done
