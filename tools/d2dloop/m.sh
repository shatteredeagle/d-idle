#!/bin/bash
# usage: m.sh TAG BOTBIN SEED HOURS   (managed d2d.lab5, lockstep, fresh Sorceress, no cheats)
# whole script (lock waits included) runs as one registered process group: tools/agentrun.sh, reap with tools/agentreap.sh
if [ -z "${AGENTRUN_ON:-}" ] && [ -x "$HOME/d2d-lab/bin/agentrun.sh" ]; then
  exec "$HOME/d2d-lab/bin/agentrun.sh" "${D2D_AGENT:-loop}" -- env AGENTRUN_ON=1 "$0" "$@"
fi
cd ~/d2d-lab/loop; T=$1
export D2BOT_INSTANCE=loop D2DSOAK_HOURS=$4
rm -rf data-$T; mkdir -p data-$T
flock -w 900 /tmp/d2d-lab.lock env D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab5 D2D_DATA=$HOME/d2d-lab/data \
 nice -n 10 ./$2 --backend d2d --managed --d2d-class 4 --char Loopy --seed $3 \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data $HOME/d2d-lab/loop/data-$T --d2d-sock $HOME/d2d-lab/loop/d2d-$T.sock \
 --addr 127.0.0.1:8475 --goals campaign --run-for 40m > bot-$T.log 2>&1
echo exit $? >> bot-$T.log
