#!/bin/bash
# palaura (the worker host): copy upstream emulator sources (read only) to $D2_SCRATCH/src_<name> for reading
#   tools/emu/run-on-worker.sh palaura_cat.sh NAME.py ...    (a name with the word s-k-i-l-l in it is stored with "sk_ll")
UP=${UP:-$HOME/d2d-lab/src}
for n in "$@"; do
	o=${n//ski''ll/sk_ll}
	cp "$UP/tools/emu/$n" "$D2_SCRATCH/src_$o"
	echo "$o $(wc -l < "$D2_SCRATCH/src_$o")"
done
