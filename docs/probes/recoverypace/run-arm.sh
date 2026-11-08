#!/bin/bash
# TASK-147: one class, fresh character, lockstep d2d (lab verbs OFF), `d2bot play`, no memory writes.
# TASK-183 A/B arm of docs/probes/classpacing/run-class.sh. usage (ON THE WORKER from the synced worktree):
# run-arm.sh ARM(old|new) TAG D2D_CLASS_INDEX SEED RUNFOR [LEVELER_BUILD]. old writes tuning.json with the pre-fix values
# (the player preset off), new leaves the player preset of `d2bot play` (ladder on) in force. Same binary for both arms.
# class index per cmd/d2dsetup/tables.json: barbarian 0, necromancer 1, paladin 2, amazon 3, sorceress 4, druid 5, assassin 6.
export PATH=$PATH:$HOME/.local/bin
ARM=$1 T=$2 C=$3 S=$4 RF=$5 LB=$6
W=$HOME/d2d-lab/recoverypace; mkdir -p $W/bin $W/logs
go build -o $W/bin/d2bot.cp ./cmd/d2bot || exit 1
cp tools/d2dslot.sh tools/agentrun.sh tools/agentreap.sh $W/bin/ 2>/dev/null
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
export D2D_AGENT=recoverypace
exec tools/agentrun.sh recoverypace --ttl 3000 -- tools/d2dslot.sh run -w 1200 -t 2400 -- bash -c '
cd '$W'; export D2BOT_INSTANCE=cp-'$T'
rm -rf data-'$T'; mkdir -p data-'$T'
if [ '$ARM' = old ]; then st=data-'$T'/instances/cp-'$T'/state; mkdir -p $st
echo "{\"version\":1,\"data\":{\"overrides\":{\"resource.recoveryEndOnKit\":0,\"resource.recoveryStallMin\":20,\"resource.recoveryStallGold\":100,\"resource.povertySellMin\":0}}}" > $st/tuning.json; fi
[ -n "'$LB'" ] && echo "{\"levelerBuild\":\"'$LB'\"}" > data-'$T'/config.json
export D2BOT_WATCH_VIDEO=0 D2BOT_WATCH_ADDR=127.0.0.1:$((9300+D2D_SLOT)) D2BOT_WATCH_ANSWER_ADDR=127.0.0.1:$((9400+D2D_SLOT)) D2BOT_WATCH_ORIGIN=http://127.0.0.1:$((9300+D2D_SLOT))
export D2D_BIN='$B' D2D_DATA=$HOME/d2d-lab/data
nice -n 19 ./bin/d2bot.cp play --backend d2d --managed --d2d-class '$C' --char Cp-'$T' --seed '$S' \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data '$W'/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for '$RF' > logs/bot-'$T'.log 2>&1
echo exit $? >> logs/bot-'$T'.log'
