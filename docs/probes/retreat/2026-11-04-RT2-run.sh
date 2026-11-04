#!/bin/bash
# runs on the worker host from the synced worktree. usage: RT2-run.sh LABEL ONLY
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab-cheats3
go build -o /tmp/retreatlive2-d2drill ./cmd/d2drill || exit 1
mkdir -p /tmp/retreatlive2-out
D2D_AGENT=retreatlive2 D2D_BIN=$B tools/agentrun.sh retreatlive2 --ttl 200 -- tools/d2dslot.sh run -n 3 -t 180 -- /tmp/retreatlive2-d2drill -group RT -only "$2" -repeat 2 -label "$1" -out /tmp/retreatlive2-out -bin $B
