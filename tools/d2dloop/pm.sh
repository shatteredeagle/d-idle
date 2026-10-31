#!/bin/bash
# usage (on the worker): pm.sh TAG BOTBIN SEED HOURS   one managed d2d run inside a slot of the pool (docs/process/worker-compute.md)
# `pm.sh par TAGPREFIX BOTBIN HOURS SEED...` starts one slot run per seed side by side and waits for all.
cd ~/d2d-lab/loop
if [ "$1" = par ]; then
 P=$2; B=$3; H=$4; shift 4
 for s in "$@"; do ./pm.sh $P$s $B $s $H & sleep 3; done; wait; exit 0
fi
T=$1 B=$2 S=$3 H=$4
exec ~/d2d-lab/d2dslot.sh run -w 3000 -- bash -c '
cd ~/d2d-lab/loop; export D2BOT_INSTANCE=loop-'$T' D2DSOAK_HOURS='$H'
rm -rf data-'$T'; mkdir -p data-'$T'
export D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab5 D2D_DATA=$HOME/d2d-lab/data
nice -n 10 ./'$B' --backend d2d --managed --d2d-class 4 --char Loopy --seed '$S' \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data $HOME/d2d-lab/loop/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for 40m > bot-'$T'.log 2>&1
echo exit $? >> bot-'$T'.log'
