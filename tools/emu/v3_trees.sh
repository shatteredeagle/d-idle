#!/bin/bash
# dmgroll3: export base 3e073888, r2 b9ecb415 (git archive from the fork) and r3 (the working tree, $1, default ~/d2d-lab/wt-mdmg-r3)
# components/ and tests/ into $D2_SCRATCH/t_LABEL (fresh each run).
#   tools/emu/run-on-worker.sh v3_trees.sh
set -u
S=${D2_SCRATCH:?}
U=$HOME/d2d-lab/fork/src
W=$HOME/d2d-lab/wt-mdmg-r3
for p in base:3e073888 r2:b9ecb415; do
	l=${p%%:*}; h=${p#*:}
	[ -d "$S/t_$l" ] && find "$S/t_$l" -delete; mkdir -p "$S/t_$l" && git -C "$U" archive "$h" components tests | tar -x -C "$S/t_$l"
done
[ -d "$S/t_r3" ] && find "$S/t_r3" -delete; mkdir -p "$S/t_r3" && cp -r "$W/components" "$W/tests" "$S/t_r3/"
echo trees ready in "$S"
