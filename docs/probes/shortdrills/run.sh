#!/bin/bash
# runs on the worker host from the synced worktree. usage: run.sh SLOT GROUP-OR-IDS LABEL
# e.g. run.sh 4 E1.merc-a shortdrill-merc
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
go build -o /tmp/shortdrills-d2drill ./cmd/d2drill && go build -o /tmp/shortdrills-d2bot ./cmd/d2bot || exit 1
mkdir -p /tmp/shortdrills-out
uptime
D2D_AGENT=shortdrills D2D_BIN=$B D2BOT_BIN=/tmp/shortdrills-d2bot tools/agentrun.sh shortdrills --ttl 1500 -- nice -n 10 tools/d2dslot.sh run -n "$1" -w 600 -t 1400 -- /tmp/shortdrills-d2drill -group E1 -only "$2" -label "$3" ${LIMIT:+-limit $LIMIT} -out /tmp/shortdrills-out -bin $B
echo "exit $?"
