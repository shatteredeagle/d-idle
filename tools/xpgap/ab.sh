#!/bin/bash
# usage (ON THE WORKER): ab.sh SEEDS... ; interleaved arms (XG_ARMS, default "off on") per seed, at most XG_MAX (default 3) of mine at once.
W=$HOME/d2d-lab/xpgap; cd $W
n=0
for s in "$@"; do for arm in ${XG_ARMS:-off on}; do
  [ $n -lt ${XG_MAX:-3} ] || { wait -n; n=$((n-1)); }
  ./run.sh ${arm}$s $arm $s 2 &
  n=$((n+1)); sleep 5
done; done
wait
