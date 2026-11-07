#!/bin/bash
# runs on the worker host from the synced worktree. usage: run-tpfirst.sh SLOT GROUP-OR-IDS LABEL
# e.g. run-tpfirst.sh 4 shortdrill-tpfirst shortdrill-tpfirst
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
go build -o /tmp/tpfirst-d2drill ./cmd/d2drill && go build -o /tmp/tpfirst-d2bot ./cmd/d2bot || exit 1
mkdir -p /tmp/tpfirst-out
uptime
D2D_AGENT=tpfirst D2D_BIN=$B D2BOT_BIN=/tmp/tpfirst-d2bot tools/agentrun.sh tpfirst --ttl 1500 -- nice -n 10 tools/d2dslot.sh run -n "$1" -w 600 -t 1400 -- /tmp/tpfirst-d2drill -group E1 -only "$2" -label "$3" -out /tmp/tpfirst-out -bin $B
echo "exit $?"
cat /tmp/tpfirst-out/*-"$3".txt 2>/dev/null | cut -c1-400
