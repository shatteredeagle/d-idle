#!/bin/bash
# runs on the worker host from the synced worktree. usage: run.sh SLOT GROUP-OR-IDS LABEL
# e.g. run.sh 4 E1.merc-a shortdrill-merc
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
go build -o /tmp/mouthfix-d2drill ./cmd/d2drill && go build -o /tmp/mouthfix-d2bot ./cmd/d2bot || exit 1
mkdir -p /tmp/mouthfix-out
uptime
D2D_AGENT=mouthfix D2D_BIN=$B D2BOT_BIN=/tmp/mouthfix-d2bot tools/agentrun.sh mouthfix --ttl 1500 -- nice -n 10 tools/d2dslot.sh run -n "$1" -w 600 -t 1400 -- /tmp/mouthfix-d2drill -group E1 -only "$2" -label "$3" ${LIMIT:+-limit $LIMIT} -out /tmp/mouthfix-out -bin $B
echo "exit $?"
