#!/bin/bash
# our own helper (not d2d code). usage: run-d2d.sh OUTDIR SEEDS "TOOL ARGS..."   e.g. run-d2d.sh out "1 2 3" "d2dspawn spawns"
# Starts one lockstep lab d2d per seed on a private socket/home, runs the attach-mode tool (SEED is appended as -seed),
# stops d2d. Run it under the lab lock:  flock -w 3000 /tmp/d2d-lab.lock run-d2d.sh ...   (lab-rules: one d2d at a time)
set -u
OUT=$1; SEEDS=$2; TOOL=$3
BIN=${D2D_BIN:-$HOME/d2d-lab/fork/bin/d2d.lab5}
mkdir -p "$OUT"; H=$OUT/home; mkdir -p $H/config $H/data
pgrep -x d2d >/dev/null && { echo "a d2d is already running, not starting"; exit 1; }
for S in $SEEDS; do
  SOCK=$H/d2d.sock; rm -f $SOCK
  XDG_CONFIG_HOME=$H/config XDG_DATA_HOME=$H/data SDL_VIDEODRIVER=dummy nice -n 10 $BIN --data $HOME/d2d-lab/data --headless \
    --devctl $SOCK --no-video --lockstep --no-save --toggle autoloot=off,autoparty=off --seed $S --start-screen ingame \
    --start-class 4 --start-name Probey > $OUT/d2d-$S.log 2>&1 &
  PID=$!
  for i in $(seq 120); do [ -S $SOCK ] && break; sleep 0.5; done
  sleep 2
  set -- $TOOL; B=$1; shift
  $HOME/d2d-lab/probes/$B -sock $SOCK -seed $S "$@" >> $OUT/$B.out 2>> $OUT/$B.err
  echo "seed $S exit $?" >> $OUT/$B.err
  kill $PID 2>/dev/null; sleep 1; kill -9 $PID 2>/dev/null; wait $PID 2>/dev/null
done
pgrep -x d2d >/dev/null && echo "WARNING a d2d is still running"
