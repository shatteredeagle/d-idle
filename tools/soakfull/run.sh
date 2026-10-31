#!/bin/bash
# usage (ON THE WORKER): run.sh TAG SEED HOURS   one managed d2d run in a slot (change soakfull). Ship bin/ (d2bot.soakfull, d2dslot.sh,
# agentrun.sh) to ~/d2d-lab/soakfull/bin first. Fresh uncheated Sorceress "Loopy-sf-SEED", lab6, run policy ON (explicit), killtime always on.
W=$HOME/d2d-lab/soakfull; cd $W || exit 2
T=$1 S=$2 H=$3
export D2D_AGENT=soakfull
exec $W/bin/d2dslot.sh run -w 3000 -- bash -c '
cd '$W'; export D2BOT_INSTANCE=sf-'$T' D2DSOAK_HOURS='$H' D2D_RUN=on
rm -rf data-'$T'; mkdir -p data-'$T'
export D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab6 D2D_DATA=$HOME/d2d-lab/data
nice -n 10 ./bin/d2bot.soakfull --backend d2d --managed --d2d-class 4 --char Loopy-sf-'$S' --seed '$S' \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data '$W'/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for 40m > logs/bot-'$T'.log 2>&1
echo exit $? >> logs/bot-'$T'.log'
