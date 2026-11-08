#!/bin/bash
# TASK-147 follow-up (panelslive): one fresh character, lockstep d2d (lab verbs OFF, zero setup writes), `d2bot play`,
# with a poller that saves the watch APIs every POLL wall seconds. ON THE WORKER from the synced worktree.
# usage: run-live.sh TAG D2D_CLASS_INDEX SEED RUNFOR [LEVELER_BUILD [ARM]]   ARM=old writes the pre-fix tuning.json (gate defaults)
# class index: barbarian 0, necromancer 1, paladin 2, amazon 3, sorceress 4, druid 5, assassin 6.
export PATH=$PATH:$HOME/.local/bin
T=$1 C=$2 S=$3 RF=$4 LB=$5 ARM=$6
W=$HOME/d2d-lab/panelslive; mkdir -p $W/bin $W/logs
go build -o $W/bin/d2bot.pl ./cmd/d2bot || exit 1
cp tools/d2dslot.sh tools/agentrun.sh tools/agentreap.sh docs/probes/panelslive/poll.sh $W/bin/ 2>/dev/null
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
export D2D_AGENT=panelslive
exec tools/agentrun.sh panelslive --ttl 3000 -- tools/d2dslot.sh run -w 1200 -t 2400 -- bash -c '
cd '$W'; export D2BOT_INSTANCE=pl-'$T'
rm -rf data-'$T' poll-'$T'; mkdir -p data-'$T'
if [ "'$ARM'" = old ]; then st=data-'$T'/instances/pl-'$T'/state; mkdir -p $st
echo "{\"version\":1,\"data\":{\"overrides\":{\"resource.recoveryEndOnKit\":0,\"resource.recoveryStallMin\":20,\"resource.recoveryStallGold\":100,\"resource.povertySellMin\":0}}}" > $st/tuning.json; fi
[ -n "'$LB'" ] && echo "{\"levelerBuild\":\"'$LB'\"}" > data-'$T'/config.json
export D2BOT_WATCH_VIDEO=0 D2BOT_WATCH_ADDR=127.0.0.1:$((9300+D2D_SLOT)) D2BOT_WATCH_ANSWER_ADDR=127.0.0.1:$((9400+D2D_SLOT)) D2BOT_WATCH_ORIGIN=http://127.0.0.1:$((9300+D2D_SLOT))
export D2D_BIN='$B' D2D_DATA=$HOME/d2d-lab/data
if [ -n "$POLL" ]; then nice -n 19 ./bin/poll.sh '$T' $((9300+D2D_SLOT)) $POLL & PP=$!; fi
nice -n 19 ./bin/d2bot.pl play --backend d2d --managed --d2d-class '$C' --char Pl-'$T' --seed '$S' \
 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s \
 --gamedir $HOME/d2d-lab/data --data '$W'/data-'$T' --d2d-sock $D2D_SOCK \
 --addr $D2D_BOT_ADDR --goals campaign --run-for '$RF' > logs/bot-'$T'.log 2>&1
echo exit $? >> logs/bot-'$T'.log
[ -n "$PP" ] && kill $PP 2>/dev/null; true'
