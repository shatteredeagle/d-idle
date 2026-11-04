#!/bin/bash
# runs on the worker from the synced worktree. usage: rl-run.sh LABEL ONLY
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab-cheats3
go build -o /tmp/retreatlive-d2drill ./cmd/d2drill || exit 1
mkdir -p /tmp/retreatlive-out
D2D_AGENT=retreatlive D2D_BIN=$B tools/agentrun.sh retreatlive --ttl 420 -- tools/d2dslot.sh run -n 3 -t 400 -- /tmp/retreatlive-d2drill -group RT -only "$2" -repeat 2 -label "$1" -out /tmp/retreatlive-out -bin $B
