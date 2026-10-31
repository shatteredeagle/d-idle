#!/bin/bash
# usage (ON THE WORKER): run.sh TAG ARM(off|on|sell) SEED HOURS   one managed d2d run in a slot (change xpgap).
# Fresh uncheated Sorceress, lab6, lockstep, run policy ON, killtime always on. ARM on writes (sell = povertySellMin only) tuning.json overrides
# for the xpgap knobs (resource.recoveryXPEarly, resource.povertySellMin) into the instance state dir; off leaves the defaults.
# Ship bin/ (d2bot.xpgap, d2dslot.sh, agentrun.sh) to ~/d2d-lab/xpgap/bin first. Wrap the driver in agentrun.sh xpgap.
W=$HOME/d2d-lab/xpgap; cd $W || exit 2
T=$1 ARM=$2 S=$3 H=$4
export D2D_AGENT=xpgap
exec $W/bin/d2dslot.sh run -w 3000 -- bash -c '
cd '$W'; export D2BOT_INSTANCE=xg-'$T' D2DSOAK_HOURS='$H' D2D_RUN=on
rm -rf data-'$T'; mkdir -p data-'$T'
if [ '$ARM' != off ]; then
  [ '$ARM' = sell ] && XG_EARLY=0
  st=data-'$T'/instances/xg-'$T'/state; mkdir -p $st
  echo "{\"version\":1,\"data\":{\"overrides\":{\"resource.recoveryXPEarly\":${XG_EARLY:-2},\"resource.povertySellMin\":${XG_SELL:-10}}}}" > $st/tuning.json
fi
export D2D_BIN=$HOME/d2d-lab/fork/bin/d2d.lab6 D2D_DATA=$HOME/d2d-lab/data
nice -n 10 ./bin/d2bot.xpgap --backend d2d --managed --d2d-class 4 --char Loopy-xg-'$S' --seed '$S' \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data '$W'/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for 40m > logs/bot-'$T'.log 2>&1
echo exit $? >> logs/bot-'$T'.log'
