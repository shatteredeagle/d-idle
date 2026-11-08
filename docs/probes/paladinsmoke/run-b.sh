#!/bin/bash
# Realtime run of TASK-177 (palpace): as run-a.sh without --lockstep, real-time d2d, ON THE WORKER from the synced worktree: fresh Paladin, lockstep d2d (lab verbs OFF: no --lab-verbs),
# `d2bot play` (ladder + watch + session record), no memory writes. usage: run-a.sh TAG SEED RUNFOR   e.g. run-a.sh pa1 7 75m
export PATH=$PATH:$HOME/.local/bin
T=$1 S=$2 RF=$3
W=$HOME/d2d-lab/paladinsmoke; mkdir -p $W/bin $W/logs
go build -o $W/bin/d2bot.pal ./cmd/d2bot || exit 1
cp tools/d2dslot.sh tools/agentrun.sh tools/agentreap.sh $W/bin/ 2>/dev/null
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
export D2D_AGENT=palmelee
exec tools/agentrun.sh palmelee --ttl 6000 -- tools/d2dslot.sh run -w 1200 -t 5400 -- bash -c '
cd '$W'; export D2BOT_INSTANCE=pal-'$T'
rm -rf data-'$T'; mkdir -p data-'$T'
export D2BOT_WATCH_VIDEO=0 D2BOT_WATCH_ADDR=127.0.0.1:$((9300+D2D_SLOT)) D2BOT_WATCH_ANSWER_ADDR=127.0.0.1:$((9400+D2D_SLOT)) D2BOT_WATCH_ORIGIN=http://127.0.0.1:$((9300+D2D_SLOT))
export D2D_BIN='$B' D2D_DATA=$HOME/d2d-lab/data
echo slot $D2D_SLOT viewer $D2BOT_WATCH_ADDR answers $D2BOT_WATCH_ANSWER_ADDR > logs/info-'$T'.txt
nice -n 10 ./bin/d2bot.pal play --backend d2d --managed --d2d-class 2 --char Pally-'$T' --seed '$S' \
 --toggle autoloot=off,autoparty=off --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data '$W'/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for '$RF' > logs/bot-'$T'.log 2>&1
echo exit $? >> logs/bot-'$T'.log'
