#!/bin/bash
# usage: ab.sh   (ON THE WORKER) 8 runs interleaved seed1-off, seed1-on, ..., at most 3 of mine at once. Own pid in pids/ab.pid.
R=$HOME/d2d-lab/runab; cd $R; mkdir -p pids; echo $$ > pids/ab.pid
alive() { n=0; for f in pids/*.pid; do [ "$f" != pids/ab.pid ] && [ -f "$f" ] && pgrep -g "$(cat $f)" >/dev/null 2>&1 && n=$((n+1)); done; echo $n; }
for s in 1 2 3 4; do for arm in off on; do
  while [ "$(alive)" -ge 3 ]; do sleep 10; done
  ./one.sh S$s$arm $arm $((40+s)) 2; sleep 5
done; done
