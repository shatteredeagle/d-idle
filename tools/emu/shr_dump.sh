#!/bin/bash
# SHRINE (decomp17, worker host): d2d's generator for Act 1 normal, N seeds: every placed object (id@x,y per room) of levels 2..37 on the game's one
# object seed, from the d2d scratch build's drlg-dump (its `game` mode, which upstream diffed against game.exe's objgroups.py: S).
#   tools/emu/run-on-worker.sh --ttl 900 shr_dump.sh FIRST LAST > dump.txt        (seeds FIRST..LAST, normal difficulty)
set -e
a=${1:?first seed}; b=${2:?last seed}
D=${D2_DATA:-$HOME/d2d-lab/data}
S=${D2_SCRATCH:-/tmp/d2agent-$USER}/mpqlink
mkdir -p "$S"
for f in "$D"/*; do ln -sf "$f" "$S/$(basename "$f")"; done
ln -sf "$D/Patch_D2.mpq" "$S/patch_d2.mpq"
export DIFFICULTY=0 ORDER=${ORDER:-shuffle} LEVELS=2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37
for s in $(seq "$a" "$b"); do
	echo "seed $s"
	nice "${DRLG:-$HOME/d2d-lab/pr/gaps/build/tools/drlg-dump/drlg-dump}" "$S" "$s" 0 game 2>/dev/null | grep -E "^(level|room|seed|open|rgn)" | grep -v "^open"
done
