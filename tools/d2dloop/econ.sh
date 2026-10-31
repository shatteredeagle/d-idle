#!/bin/bash
# usage: econ.sh build|run TAG SEED HOURS   (run from a worktree on the head box; syncs it to the worker, builds cmd/d2bot there,
# runs the managed d2d loop of m.sh with that binary). d2d runs hold /tmp/d2d-lab.lock (m.sh does).
set -e
BIN=${BIN:-d2bot.econ}
. "$(dirname "$0")/../env.sh"
O=$D2_WORKER_USER@$D2_WORKER_HOST
W=$(git rev-parse --show-toplevel)
case $1 in
 build)
  rsync -a --delete --exclude .git --exclude '*.test' "$W"/ $O:d2d-lab/econ/src/
  ssh $O "cd ~/d2d-lab/econ/src && go build -o ~/d2d-lab/loop/$BIN ./cmd/d2bot && go build -o ~/d2d-lab/loop/d2dsoak.econ ./cmd/d2dsoak && echo built" ;;
 run)
  scp -q "$W"/tools/d2dloop/*.sh "$W"/tools/d2dloop/*.py $O:d2d-lab/loop/
  ssh $O "mkdir -p ~/d2d-lab/bin" && scp -q "$W"/tools/agentrun.sh "$W"/tools/agentreap.sh $O:d2d-lab/bin/
  ssh -f $O "cd ~/d2d-lab/loop && chmod +x m.sh && nohup ./m.sh $2 $BIN $3 $4 >/dev/null 2>&1 </dev/null &" ; echo started $2 ;;
esac
