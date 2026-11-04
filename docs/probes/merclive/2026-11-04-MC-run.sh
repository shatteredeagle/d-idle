#!/bin/bash
# runs on the worker host from the synced worktree. usage: MC-run.sh LABEL ONLY
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab-cheats3
go build -o /tmp/merclive-d2drill ./cmd/d2drill || exit 1
mkdir -p /tmp/merclive-out
D2D_AGENT=merclive D2D_BIN=$B tools/agentrun.sh merclive --ttl 210 -- tools/d2dslot.sh run -n 3 -t 200 -- /tmp/merclive-d2drill -group D8 -only "$2" -repeat 1 -label "$1" -out /tmp/merclive-out -bin $B
